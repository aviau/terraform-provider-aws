package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53recoverycontrolconfig"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSRoute53RecoveryControlConfigRoutingControl_basic(t *testing.T) {
	rClusterName := acctest.RandomWithPrefix("tf-acc-test-cluster")
	rRoutingControlName := acctest.RandomWithPrefix("tf-acc-test-routing-control")
	resourceName := "aws_route53recoverycontrolconfig_routing_control.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, route53recoverycontrolconfig.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsRoute53RecoveryControlConfigRoutingControlDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsRoute53RecoveryControlConfigRoutingControlConfig_InDefaultControlPanel(rClusterName, rRoutingControlName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsRoute53RecoveryControlConfigRoutingControlExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", rRoutingControlName),
					resource.TestCheckResourceAttr(resourceName, "status", "DEPLOYED"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSRoute53RecoveryControlConfigRoutingControl_NonDefaultControlPanel(t *testing.T) {
	rClusterName := acctest.RandomWithPrefix("tf-acc-test-cluster")
	rControlPanelName := acctest.RandomWithPrefix("tf-acc-test-control-panel")
	rRoutingControlName := acctest.RandomWithPrefix("tf-acc-test-routing-control")
	resourceName := "aws_route53recoverycontrolconfig_routing_control.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, route53recoverycontrolconfig.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsRoute53RecoveryControlConfigRoutingControlDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsRoute53RecoveryControlConfigRoutingControlConfig_InNonDefaultControlPanel(rClusterName, rControlPanelName, rRoutingControlName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsRoute53RecoveryControlConfigRoutingControlExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", rRoutingControlName),
					resource.TestCheckResourceAttr(resourceName, "status", "DEPLOYED"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAwsRoute53RecoveryControlConfigRoutingControlDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).route53recoverycontrolconfigconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53recoverycontrolconfig_routing_control" {
			continue
		}

		input := &route53recoverycontrolconfig.DescribeRoutingControlInput{
			RoutingControlArn: aws.String(rs.Primary.ID),
		}

		_, err := conn.DescribeRoutingControl(input)

		if err == nil {
			return fmt.Errorf("Route53RecoveryControlConfig Routing Control (%s) not deleted", rs.Primary.ID)
		}
	}

	return nil
}

func testAccAwsRoute53RecoveryControlConfigClusterBase(rName string) string {
	return fmt.Sprintf(`
resource "aws_route53recoverycontrolconfig_cluster" "test" {
  name = %[1]q
}
`, rName)
}

func testAccAwsRoute53RecoveryControlConfigRoutingControlConfig_InDefaultControlPanel(rName, rName2 string) string {
	return composeConfig(
		testAccAwsRoute53RecoveryControlConfigClusterBase(rName), fmt.Sprintf(`
resource "aws_route53recoverycontrolconfig_routing_control" "test" {
  name        = %q
  cluster_arn = aws_route53recoverycontrolconfig_cluster.test.cluster_arn
}
`, rName2))
}

func testAccAwsRoute53RecoveryControlConfigControlPanelBase(rName string) string {
	return fmt.Sprintf(`
resource "aws_route53recoverycontrolconfig_control_panel" "test" {
  name        = %q
  cluster_arn = aws_route53recoverycontrolconfig_cluster.test.cluster_arn
}
`, rName)
}

func testAccAwsRoute53RecoveryControlConfigRoutingControlConfig_InNonDefaultControlPanel(rName, rName2, rName3 string) string {
	return composeConfig(
		testAccAwsRoute53RecoveryControlConfigClusterBase(rName),
		testAccAwsRoute53RecoveryControlConfigControlPanelBase(rName2),
		fmt.Sprintf(`
resource "aws_route53recoverycontrolconfig_routing_control" "test" {
  name              = %q
  cluster_arn       = aws_route53recoverycontrolconfig_cluster.test.cluster_arn
  control_panel_arn = aws_route53recoverycontrolconfig_control_panel.test.control_panel_arn
}
`, rName3))
}

func testAccCheckAwsRoute53RecoveryControlConfigRoutingControlExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).route53recoverycontrolconfigconn

		input := &route53recoverycontrolconfig.DescribeRoutingControlInput{
			RoutingControlArn: aws.String(rs.Primary.ID),
		}

		_, err := conn.DescribeRoutingControl(input)

		return err
	}
}
