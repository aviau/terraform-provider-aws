package route53_test

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfroute53 "github.com/hashicorp/terraform-provider-aws/internal/service/route53"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestCleanZoneID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Input, Output string
	}{
		{"/hostedzone/foo", "foo"},
		{"/change/foo", "/change/foo"},
		{"/bar", "/bar"},
	}

	for _, tc := range cases {
		actual := tfroute53.CleanZoneID(tc.Input)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestCleanChangeID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Input, Output string
	}{
		{"/hostedzone/foo", "/hostedzone/foo"},
		{"/change/foo", "foo"},
		{"/bar", "/bar"},
	}

	for _, tc := range cases {
		actual := tfroute53.CleanChangeID(tc.Input)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestTrimTrailingPeriod(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Input  interface{}
		Output string
	}{
		{"example.com", "example.com"},
		{"example.com.", "example.com"},
		{"www.example.com.", "www.example.com"},
		{"", ""},
		{".", "."},
		{aws.String("example.com"), "example.com"},
		{aws.String("example.com."), "example.com"},
		{(*string)(nil), ""},
		{42, ""},
		{nil, ""},
	}

	for _, tc := range cases {
		actual := tfroute53.TrimTrailingPeriod(tc.Input)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

// add sweeper to delete resources

func TestAccRoute53Zone_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_basic(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					acctest.MatchResourceAttrGlobalARNNoAccount(resourceName, "arn", "route53", regexp.MustCompile("hostedzone/.+")),
					resource.TestCheckResourceAttr(resourceName, "name", zoneName),
					resource.TestCheckResourceAttr(resourceName, "name_servers.#", "4"),
					resource.TestCheckResourceAttrSet(resourceName, "primary_name_server"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "vpc.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
		},
	})
}

func TestAccRoute53Zone_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_basic(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfroute53.ResourceZone(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccRoute53Zone_multiple(t *testing.T) {
	ctx := acctest.Context(t)
	var zone0, zone1, zone2, zone3, zone4 route53.GetHostedZoneOutput
	domainName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_multiple(domainName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, "aws_route53_zone.test.0", &zone0),
					testAccCheckDomainName(&zone0, fmt.Sprintf("subdomain0.%s.", domainName)),
					testAccCheckZoneExists(ctx, "aws_route53_zone.test.1", &zone1),
					testAccCheckDomainName(&zone1, fmt.Sprintf("subdomain1.%s.", domainName)),
					testAccCheckZoneExists(ctx, "aws_route53_zone.test.2", &zone2),
					testAccCheckDomainName(&zone2, fmt.Sprintf("subdomain2.%s.", domainName)),
					testAccCheckZoneExists(ctx, "aws_route53_zone.test.3", &zone3),
					testAccCheckDomainName(&zone3, fmt.Sprintf("subdomain3.%s.", domainName)),
					testAccCheckZoneExists(ctx, "aws_route53_zone.test.4", &zone4),
					testAccCheckDomainName(&zone4, fmt.Sprintf("subdomain4.%s.", domainName)),
				),
			},
		},
	})
}

func TestAccRoute53Zone_comment(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_comment(zoneName, "comment1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "comment", "comment1"),
				),
			},
			{
				Config: testAccZoneConfig_comment(zoneName, "comment2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "comment", "comment2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
		},
	})
}

func TestAccRoute53Zone_delegationSetID(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	delegationSetResourceName := "aws_route53_delegation_set.test"
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_delegationSetID(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttrPair(resourceName, "delegation_set_id", delegationSetResourceName, "id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
		},
	})
}

func TestAccRoute53Zone_forceDestroy(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_forceDestroy(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					// Add >100 records to verify pagination works ok
					testAccCreateRandomRecordsInZoneID(ctx, &zone, 100),
					testAccCreateRandomRecordsInZoneID(ctx, &zone, 5),
				),
			},
		},
	})
}

func TestAccRoute53Zone_ForceDestroy_trailingPeriod(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_forceDestroyTrailingPeriod(zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					// Add >100 records to verify pagination works ok
					testAccCreateRandomRecordsInZoneID(ctx, &zone, 100),
					testAccCreateRandomRecordsInZoneID(ctx, &zone, 5),
				),
			},
		},
	})
}

func TestAccRoute53Zone_tags(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	resourceName := "aws_route53_zone.test"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_tags1(zoneName, "tag1key", "tag1value"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.tag1key", "tag1value"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
			{
				Config: testAccZoneConfig_tags2(zoneName, "tag1key", "tag1valueupdated", "tag2key", "tag2value"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.tag1key", "tag1valueupdated"),
					resource.TestCheckResourceAttr(resourceName, "tags.tag2key", "tag2value"),
				),
			},
			{
				Config: testAccZoneConfig_tags1(zoneName, "tag2key", "tag2value"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.tag2key", "tag2value"),
				),
			},
		},
	})
}

func TestAccRoute53Zone_VPC_single(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_route53_zone.test"
	vpcResourceName := "aws_vpc.test1"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_vpcSingle(rName, zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "vpc.#", "1"),
					testAccCheckZoneAssociatesVPC(vpcResourceName, &zone),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
		},
	})
}

func TestAccRoute53Zone_VPC_multiple(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_route53_zone.test"
	vpcResourceName1 := "aws_vpc.test1"
	vpcResourceName2 := "aws_vpc.test2"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_vpcMultiple(rName, zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "vpc.#", "2"),
					testAccCheckZoneAssociatesVPC(vpcResourceName1, &zone),
					testAccCheckZoneAssociatesVPC(vpcResourceName2, &zone),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_destroy"},
			},
		},
	})
}

func TestAccRoute53Zone_VPC_updates(t *testing.T) {
	ctx := acctest.Context(t)
	var zone route53.GetHostedZoneOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_route53_zone.test"
	vpcResourceName1 := "aws_vpc.test1"
	vpcResourceName2 := "aws_vpc.test2"
	zoneName := acctest.RandomDomainName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, route53.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckZoneDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccZoneConfig_vpcSingle(rName, zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "vpc.#", "1"),
					testAccCheckZoneAssociatesVPC(vpcResourceName1, &zone),
				),
			},
			{
				Config: testAccZoneConfig_vpcMultiple(rName, zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "vpc.#", "2"),
					testAccCheckZoneAssociatesVPC(vpcResourceName1, &zone),
					testAccCheckZoneAssociatesVPC(vpcResourceName2, &zone),
				),
			},
			{
				Config: testAccZoneConfig_vpcSingle(rName, zoneName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneExists(ctx, resourceName, &zone),
					resource.TestCheckResourceAttr(resourceName, "vpc.#", "1"),
					testAccCheckZoneAssociatesVPC(vpcResourceName1, &zone),
				),
			},
		},
	})
}

func testAccCheckZoneDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).Route53Conn(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_route53_zone" {
				continue
			}

			_, err := tfroute53.FindHostedZoneByID(ctx, conn, rs.Primary.ID)

			if tfresource.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("Route53 Hosted Zone %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCreateRandomRecordsInZoneID(ctx context.Context, zone *route53.GetHostedZoneOutput, recordsCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).Route53Conn(ctx)

		var changes []*route53.Change
		if recordsCount > 100 {
			return fmt.Errorf("Route53 API only allows 100 record sets in a single batch")
		}
		for i := 0; i < recordsCount; i++ {
			changes = append(changes, &route53.Change{
				Action: aws.String("UPSERT"),
				ResourceRecordSet: &route53.ResourceRecordSet{
					Name: aws.String(fmt.Sprintf("%d-tf-acc-random.%s", sdkacctest.RandInt(), *zone.HostedZone.Name)),
					Type: aws.String("CNAME"),
					ResourceRecords: []*route53.ResourceRecord{
						{Value: aws.String(fmt.Sprintf("random.%s", *zone.HostedZone.Name))},
					},
					TTL: aws.Int64(30),
				},
			})
		}

		req := &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: zone.HostedZone.Id,
			ChangeBatch: &route53.ChangeBatch{
				Comment: aws.String("Generated by Terraform"),
				Changes: changes,
			},
		}
		log.Printf("[DEBUG] Change set: %s\n", *req)
		changeInfo, err := tfroute53.ChangeResourceRecordSets(ctx, conn, req)
		if err != nil {
			return err
		}
		err = tfroute53.WaitForRecordSetToSync(ctx, conn, tfroute53.CleanChangeID(*changeInfo.Id))
		return err
	}
}

func testAccCheckZoneExists(ctx context.Context, n string, v *route53.GetHostedZoneOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Route53 Hosted Zone ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).Route53Conn(ctx)

		output, err := tfroute53.FindHostedZoneByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckZoneAssociatesVPC(n string, zone *route53.GetHostedZoneOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		for _, vpc := range zone.VPCs {
			if aws.StringValue(vpc.VPCId) == rs.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("VPC: %s is not associated to Zone: %v", n, tfroute53.CleanZoneID(aws.StringValue(zone.HostedZone.Id)))
	}
}

func testAccCheckDomainName(zone *route53.GetHostedZoneOutput, domain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.HostedZone.Name == nil {
			return fmt.Errorf("Empty name in HostedZone for domain %s", domain)
		}

		if aws.StringValue(zone.HostedZone.Name) == domain {
			return nil
		}

		return fmt.Errorf("Invalid domain name. Expected %s is %s", domain, *zone.HostedZone.Name)
	}
}

func testAccZoneConfig_basic(zoneName string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  name = "%[1]s."
}
`, zoneName)
}

func testAccZoneConfig_multiple(domainName string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  count = 5

  name = "subdomain${count.index}.%[1]s"
}
`, domainName)
}

func testAccZoneConfig_comment(zoneName, comment string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  comment = %[1]q
  name    = "%[2]s."
}
`, comment, zoneName)
}

func testAccZoneConfig_delegationSetID(zoneName string) string {
	return fmt.Sprintf(`
resource "aws_route53_delegation_set" "test" {}

resource "aws_route53_zone" "test" {
  delegation_set_id = aws_route53_delegation_set.test.id
  name              = "%[1]s."
}
`, zoneName)
}

func testAccZoneConfig_forceDestroy(zoneName string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  force_destroy = true
  name          = "%[1]s"
}
`, zoneName)
}

func testAccZoneConfig_forceDestroyTrailingPeriod(zoneName string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  force_destroy = true
  name          = "%[1]s."
}
`, zoneName)
}

func testAccZoneConfig_tags1(zoneName, tag1Key, tag1Value string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  name = "%[1]s."

  tags = {
    %[2]q = %[3]q
  }
}
`, zoneName, tag1Key, tag1Value)
}

func testAccZoneConfig_tags2(zoneName, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return fmt.Sprintf(`
resource "aws_route53_zone" "test" {
  name = "%[1]s."

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, zoneName, tag1Key, tag1Value, tag2Key, tag2Value)
}

func testAccZoneConfig_vpcSingle(rName, zoneName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test1" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_route53_zone" "test" {
  name = "%[2]s."

  vpc {
    vpc_id = aws_vpc.test1.id
  }
}
`, rName, zoneName)
}

func testAccZoneConfig_vpcMultiple(rName, zoneName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test1" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc" "test2" {
  cidr_block = "10.2.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_route53_zone" "test" {
  name = "%[2]s."

  vpc {
    vpc_id = aws_vpc.test1.id
  }

  vpc {
    vpc_id = aws_vpc.test2.id
  }
}
`, rName, zoneName)
}
