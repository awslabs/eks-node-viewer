/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/pricing/pricingiface"
	"go.uber.org/multierr"

	"github.com/awslabs/eks-node-viewer/pkg/model"
	nvp "github.com/awslabs/eks-node-viewer/pkg/pricing"
)

type pricingProvider struct {
	ec2     ec2iface.EC2API
	pricing pricingiface.PricingAPI
	region  string

	mu                      sync.RWMutex
	onUpdateFuncs           []func()
	onDemandPrices          map[ec2types.InstanceType]float64
	spotPrices              map[ec2types.InstanceType]zonalPricing
	fargateVCPUPricePerHour float64
	fargateGBPricePerHour   float64
}

func (p *pricingProvider) OnUpdate(onUpdate func()) {
	p.onUpdateFuncs = append(p.onUpdateFuncs, onUpdate)
}

func (p *pricingProvider) NodePrice(n *model.Node) (float64, bool) {
	if n.IsOnDemand() {
		if price, ok := p.OnDemandPrice(n.InstanceType()); ok {
			return price, true
		}
	} else if n.IsSpot() {
		if price, ok := p.SpotPrice(n.InstanceType(), n.Zone()); ok {
			return price, true
		}
	} else if n.IsFargate() && len(n.Pods()) == 1 {
		cpu, mem, ok := n.Pods()[0].FargateCapacityProvisioned()
		if ok {
			if price, ok := p.FargatePrice(cpu, mem); ok {
				return price, true
			}
		}
	}
	return math.NaN(), false
}

// zonalPricing is used to capture the per-zone price
// for spot data as well as the default price
// based on on-demand price when the controller first
// comes up
type zonalPricing struct {
	defaultPrice float64 // Used until we get the spot pricing data
	prices       map[string]float64
}

func newZonalPricing(defaultPrice float64) zonalPricing {
	z := zonalPricing{
		prices: map[string]float64{},
	}
	z.defaultPrice = defaultPrice
	return z
}

// pricingUpdatePeriod is how often we try to update our pricing information after the initial update on startup
const pricingUpdatePeriod = 12 * time.Hour

// NewPricingAPI returns a pricing API configured based on a particular region
func NewPricingAPI(sess *session.Session, region string) pricingiface.PricingAPI {
	if sess == nil {
		return nil
	}
	// pricing API doesn't have an endpoint in all regions
	pricingAPIRegion := "us-east-1"
	if strings.HasPrefix(region, "ap-") {
		pricingAPIRegion = "ap-south-1"
	} else if strings.HasPrefix(region, "cn-") {
		pricingAPIRegion = "cn-northwest-1"
	} else if strings.HasPrefix(region, "eu-") {
		pricingAPIRegion = "eu-central-1"
	}
	return pricing.New(sess, &aws.Config{Region: aws.String(pricingAPIRegion)})
}

var allPrices = []map[string]map[ec2types.InstanceType]float64{
	InitialOnDemandPricesAWS,
	InitialOnDemandPricesUSGov,
	InitialOnDemandPricesCN,
}

func getStaticPrices(region string) map[ec2types.InstanceType]float64 {
	for _, priceSet := range allPrices {
		if prices, ok := priceSet[region]; ok {
			return prices
		}
	}
	return InitialOnDemandPricesAWS["us-east-1"]
}

func NewStaticPricingProvider() nvp.Provider {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	return &pricingProvider{
		onDemandPrices: getStaticPrices(region),
		spotPrices:     map[ec2types.InstanceType]zonalPricing{},
	}
}

func NewPricingProvider(ctx context.Context, sess *session.Session) nvp.Provider {
	region := "us-west-2"
	if aws.StringValue(sess.Config.Region) != "" {
		region = aws.StringValue(sess.Config.Region)
	}
	p := &pricingProvider{
		region:         region,
		onDemandPrices: getStaticPrices(region),
		spotPrices:     map[ec2types.InstanceType]zonalPricing{},
		ec2:            ec2.New(sess),
		pricing:        NewPricingAPI(sess, region),
	}

	go func() {
		// perform an initial price update at startup
		p.updatePricing(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pricingUpdatePeriod):
				p.updatePricing(ctx)
			}
		}
	}()
	return p
}

// OnDemandPrice returns the last known on-demand price for a given instance type, returning an error if there is no
// known on-demand pricing for the instance type.
func (p *pricingProvider) OnDemandPrice(instanceType ec2types.InstanceType) (float64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	price, ok := p.onDemandPrices[instanceType]
	if !ok {
		return 0.0, false
	}
	return price, true
}

func (p *pricingProvider) FargatePrice(cpu, memory float64) (float64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.fargateGBPricePerHour == 0 || p.fargateVCPUPricePerHour == 0 {
		return 0, false
	}
	return cpu*p.fargateVCPUPricePerHour + memory*p.fargateGBPricePerHour, true
}

// TODO: method to fetch attached EBS prices

// SpotPrice returns the last known spot price for a given instance type and zone, returning an error
// if there is no known spot pricing for that instance type or zone
func (p *pricingProvider) SpotPrice(instanceType ec2types.InstanceType, zone string) (float64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if val, ok := p.spotPrices[instanceType]; ok {
		if price, ok := p.spotPrices[instanceType].prices[zone]; ok {
			return price, true
		}
		return val.defaultPrice, true
	}
	return 0.0, false
}

func (p *pricingProvider) updatePricing(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.updateOnDemandPricing(ctx); err != nil {
			log.Printf("updating on-demand pricing, %s, using existing pricing data", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.updateSpotPricing(ctx); err != nil {
			log.Printf("updating spot pricing, %s, using existing pricing data", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.updateFargatePricing(ctx); err != nil {
			log.Printf("updating fargate pricing, %s", err)
		}
	}()
	wg.Wait()

	// notify anyone that cares
	for _, f := range p.onUpdateFuncs {
		f()
	}
}

func (p *pricingProvider) updateOnDemandPricing(ctx context.Context) error {
	// standard on-demand instances
	var wg sync.WaitGroup
	var onDemandPrices, onDemandMetalPrices map[ec2types.InstanceType]float64
	var onDemandErr, onDemandMetalErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		onDemandPrices, onDemandErr = p.fetchOnDemandPricing(ctx,
			&pricing.Filter{
				Field: aws.String("tenancy"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Shared"),
			},
			&pricing.Filter{
				Field: aws.String("productFamily"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Compute Instance"),
			})
	}()

	// bare metal on-demand prices
	wg.Add(1)
	go func() {
		defer wg.Done()
		onDemandMetalPrices, onDemandMetalErr = p.fetchOnDemandPricing(ctx,
			&pricing.Filter{
				Field: aws.String("tenancy"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Dedicated"),
			},
			&pricing.Filter{
				Field: aws.String("productFamily"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Compute Instance (bare metal)"),
			})
	}()

	wg.Wait()
	err := multierr.Append(onDemandErr, onDemandMetalErr)
	if err != nil {
		return err
	}

	if len(onDemandPrices) == 0 || len(onDemandMetalPrices) == 0 {
		return errors.New("no on-demand pricing found")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.onDemandPrices = map[ec2types.InstanceType]float64{}
	for _, m := range []map[ec2types.InstanceType]float64{onDemandPrices, onDemandMetalPrices} {
		for k, v := range m {
			p.onDemandPrices[k] = v
		}
	}
	return nil
}

func (p *pricingProvider) fetchOnDemandPricing(ctx context.Context, additionalFilters ...*pricing.Filter) (map[ec2types.InstanceType]float64, error) {
	prices := map[ec2types.InstanceType]float64{}
	filters := append([]*pricing.Filter{
		{
			Field: aws.String("regionCode"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String(p.region),
		},
		{
			Field: aws.String("serviceCode"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String("AmazonEC2"),
		},
		{
			Field: aws.String("preInstalledSw"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String("NA"),
		},
		{
			Field: aws.String("operatingSystem"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String("Linux"),
		},
		{
			Field: aws.String("capacitystatus"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String("Used"),
		},
		{
			Field: aws.String("marketoption"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String("OnDemand"),
		}},
		additionalFilters...)
	if err := p.pricing.GetProductsPagesWithContext(ctx, &pricing.GetProductsInput{
		Filters:     filters,
		ServiceCode: aws.String("AmazonEC2")}, p.onDemandPage(prices)); err != nil {
		return nil, err
	}
	return prices, nil
}

// turning off cyclo here, it measures as a 12 due to all of the type checks of the pricing data which returns a deeply
// nested map[string]interface{}
// nolint: gocyclo
func (p *pricingProvider) onDemandPage(prices map[ec2types.InstanceType]float64) func(output *pricing.GetProductsOutput, b bool) bool {
	// this isn't the full pricing struct, just the portions we care about
	type priceItem struct {
		Product struct {
			Attributes struct {
				InstanceType string
			}
		}
		Terms struct {
			OnDemand map[string]struct {
				PriceDimensions map[string]struct {
					PricePerUnit map[string]string
				}
			}
		}
	}

	return func(output *pricing.GetProductsOutput, b bool) bool {
		currency := "USD"
		if strings.HasPrefix(p.region, "cn-") {
			currency = "CNY"
		}
		for _, outer := range output.PriceList {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			if err := enc.Encode(outer); err != nil {
				log.Printf("encoding %s", err)
			}
			dec := json.NewDecoder(&buf)
			var pItem priceItem
			if err := dec.Decode(&pItem); err != nil {
				log.Printf("decoding %s", err)
			}
			if pItem.Product.Attributes.InstanceType == "" {
				continue
			}
			for _, term := range pItem.Terms.OnDemand {
				for _, v := range term.PriceDimensions {
					price, err := strconv.ParseFloat(v.PricePerUnit[currency], 64)
					if err != nil || price == 0 {
						continue
					}
					prices[ec2types.InstanceType(pItem.Product.Attributes.InstanceType)] = price
				}
			}
		}
		return true
	}
}

// nolint: gocyclo
func (p *pricingProvider) updateSpotPricing(ctx context.Context) error {
	totalOfferings := 0

	prices := map[ec2types.InstanceType]map[string]float64{}
	if err := p.ec2.DescribeSpotPriceHistoryPagesWithContext(ctx, &ec2.DescribeSpotPriceHistoryInput{
		ProductDescriptions: []*string{aws.String("Linux/UNIX"), aws.String("Linux/UNIX (Amazon VPC)")},
		// get the latest spot price for each instance type
		StartTime: aws.Time(time.Now()),
	}, func(output *ec2.DescribeSpotPriceHistoryOutput, b bool) bool {
		for _, sph := range output.SpotPriceHistory {
			spotPriceStr := aws.StringValue(sph.SpotPrice)
			spotPrice, err := strconv.ParseFloat(spotPriceStr, 64)
			// these errors shouldn't occur, but if pricing API does have an error, we ignore the record
			if err != nil {
				log.Printf("unable to parse price record %#v", sph)
				continue
			}
			if sph.Timestamp == nil {
				continue
			}
			instanceType := ec2types.InstanceType(aws.StringValue(sph.InstanceType))
			az := aws.StringValue(sph.AvailabilityZone)
			_, ok := prices[instanceType]
			if !ok {
				prices[instanceType] = map[string]float64{}
			}
			prices[instanceType][az] = spotPrice
		}
		return true
	}); err != nil {
		return err
	}
	if len(prices) == 0 {
		return errors.New("no spot pricing found")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	for it, zoneData := range prices {
		if _, ok := p.spotPrices[it]; !ok {
			p.spotPrices[it] = newZonalPricing(0)
		}
		for zone, price := range zoneData {
			p.spotPrices[it].prices[zone] = price
		}
		totalOfferings += len(zoneData)
	}
	return nil
}

func (p *pricingProvider) updateFargatePricing(ctx context.Context) error {
	filters := []*pricing.Filter{
		{
			Field: aws.String("regionCode"),
			Type:  aws.String("TERM_MATCH"),
			Value: aws.String(p.region),
		},
	}
	if err := p.pricing.GetProductsPagesWithContext(ctx, &pricing.GetProductsInput{
		Filters:     filters,
		ServiceCode: aws.String("AmazonEKS")}, p.fargatePage); err != nil {
		return err
	}
	return nil
}

func (p *pricingProvider) fargatePage(output *pricing.GetProductsOutput, _ bool) bool {
	// this isn't the full pricing struct, just the portions we care about
	type priceItem struct {
		Product struct {
			ProductFamily string
			Attributes    struct {
				UsageType  string
				MemoryType string
			}
		}
		Terms struct {
			OnDemand map[string]struct {
				PriceDimensions map[string]struct {
					PricePerUnit struct {
						USD string
					}
				}
			}
		}
	}

	for _, outer := range output.PriceList {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(outer); err != nil {
			log.Printf("encoding %s", err)
		}
		dec := json.NewDecoder(&buf)
		var pItem priceItem
		if err := dec.Decode(&pItem); err != nil {
			log.Printf("decoding %s", err)
		}
		if !strings.Contains(pItem.Product.Attributes.UsageType, "Fargate") {
			continue
		}
		name := pItem.Product.Attributes.UsageType
		for _, term := range pItem.Terms.OnDemand {
			for _, v := range term.PriceDimensions {
				price, err := strconv.ParseFloat(v.PricePerUnit.USD, 64)
				if err != nil || price == 0 {
					continue
				}
				if strings.Contains(name, "vCPU-Hours") {
					p.mu.Lock()
					p.fargateVCPUPricePerHour = price
					p.mu.Unlock()
				} else if strings.Contains(name, "GB-Hours") {
					p.mu.Lock()
					p.fargateGBPricePerHour = price
					p.mu.Unlock()
				} else {
					log.Println("unsupported fargate price information found", name)
				}
			}
		}
	}
	return true

}
