package cur_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	cur "github.com/aws/aws-sdk-go/service/costandusagereportservice"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func testAccReportDefinitionDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_cur_report_definition.test"
	datasourceName := "data.aws_cur_report_definition.test"

	reportName := sdkacctest.RandomWithPrefix("tf_acc_test")
	bucketName := fmt.Sprintf("tf-test-bucket-%d", sdkacctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckRegion(t, endpoints.UsEast1RegionID) },
		ErrorCheck:               acctest.ErrorCheck(t, cur.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckReportDefinitionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccReportDefinitionDataSourceConfig_basic(reportName, bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "report_name", resourceName, "report_name"),
					resource.TestCheckResourceAttrPair(datasourceName, "time_unit", resourceName, "time_unit"),
					resource.TestCheckResourceAttrPair(datasourceName, "compression", resourceName, "compression"),
					resource.TestCheckResourceAttrPair(datasourceName, "additional_schema_elements.#", resourceName, "additional_schema_elements.#"),
					resource.TestCheckResourceAttrPair(datasourceName, "s3_bucket", resourceName, "s3_bucket"),
					resource.TestCheckResourceAttrPair(datasourceName, "s3_prefix", resourceName, "s3_prefix"),
					resource.TestCheckResourceAttrPair(datasourceName, "s3_region", resourceName, "s3_region"),
					resource.TestCheckResourceAttrPair(datasourceName, "additional_artifacts.#", resourceName, "additional_artifacts.#"),
				),
			},
		},
	})
}

func testAccReportDefinitionDataSource_additional(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_cur_report_definition.test"
	datasourceName := "data.aws_cur_report_definition.test"

	reportName := sdkacctest.RandomWithPrefix("tf_acc_test")
	bucketName := fmt.Sprintf("tf-test-bucket-%d", sdkacctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckRegion(t, endpoints.UsEast1RegionID) },
		ErrorCheck:               acctest.ErrorCheck(t, cur.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckReportDefinitionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccReportDefinitionDataSourceConfig_additional(reportName, bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "report_name", resourceName, "report_name"),
					resource.TestCheckResourceAttrPair(datasourceName, "time_unit", resourceName, "time_unit"),
					resource.TestCheckResourceAttrPair(datasourceName, "compression", resourceName, "compression"),
					resource.TestCheckResourceAttrPair(datasourceName, "additional_schema_elements.#", resourceName, "additional_schema_elements.#"),
					resource.TestCheckResourceAttrPair(datasourceName, "s3_bucket", resourceName, "s3_bucket"),
					resource.TestCheckResourceAttrPair(datasourceName, "s3_prefix", resourceName, "s3_prefix"),
					resource.TestCheckResourceAttrPair(datasourceName, "s3_region", resourceName, "s3_region"),
					resource.TestCheckResourceAttrPair(datasourceName, "additional_artifacts.#", resourceName, "additional_artifacts.#"),
					resource.TestCheckResourceAttrPair(datasourceName, "refresh_closed_reports", resourceName, "refresh_closed_reports"),
					resource.TestCheckResourceAttrPair(datasourceName, "report_versioning", resourceName, "report_versioning"),
				),
			},
		},
	})
}

func testAccReportDefinitionDataSourceConfig_basic(reportName string, bucketName string) string {
	return fmt.Sprintf(`
data "aws_billing_service_account" "test" {}

data "aws_partition" "current" {}

resource "aws_s3_bucket" "test" {
  bucket        = %[2]q
  force_destroy = true
}

resource "aws_s3_bucket_policy" "test" {
  bucket = aws_s3_bucket.test.id

  policy = <<POLICY
{
  "Version": "2008-10-17",
  "Id": "s3policy",
  "Statement": [
    {
      "Sid": "AllowCURBillingACLPolicy",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${data.aws_billing_service_account.test.arn}"
      },
      "Action": [
        "s3:GetBucketAcl",
        "s3:GetBucketPolicy"
      ],
      "Resource": "${aws_s3_bucket.test.arn}"
    },
    {
      "Sid": "AllowCURPutObject",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${data.aws_billing_service_account.test.arn}"
      },
      "Action": "s3:PutObject",
      "Resource": "arn:${data.aws_partition.current.partition}:s3:::${aws_s3_bucket.test.id}/*"
    }
  ]
}
POLICY
}

resource "aws_cur_report_definition" "test" {
  depends_on = [aws_s3_bucket_policy.test] # needed to avoid "ValidationException: Failed to verify customer bucket permission."

  report_name                = %[1]q
  time_unit                  = "DAILY"
  format                     = "textORcsv"
  compression                = "GZIP"
  additional_schema_elements = ["RESOURCES", "SPLIT_COST_ALLOCATION_DATA"]
  s3_bucket                  = aws_s3_bucket.test.id
  s3_prefix                  = ""
  s3_region                  = aws_s3_bucket.test.region
  additional_artifacts       = ["REDSHIFT", "QUICKSIGHT"]
}

data "aws_cur_report_definition" "test" {
  report_name = aws_cur_report_definition.test.report_name
}
`, reportName, bucketName)
}

func testAccReportDefinitionDataSourceConfig_additional(reportName string, bucketName string) string {
	return fmt.Sprintf(`
data "aws_billing_service_account" "test" {}

data "aws_partition" "current" {}

resource "aws_s3_bucket" "test" {
  bucket        = %[2]q
  force_destroy = true
}

resource "aws_s3_bucket_policy" "test" {
  bucket = aws_s3_bucket.test.id

  policy = <<POLICY
{
  "Version": "2008-10-17",
  "Id": "s3policy",
  "Statement": [
    {
      "Sid": "AllowCURBillingACLPolicy",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${data.aws_billing_service_account.test.arn}"
      },
      "Action": [
        "s3:GetBucketAcl",
        "s3:GetBucketPolicy"
      ],
      "Resource": "${aws_s3_bucket.test.arn}"
    },
    {
      "Sid": "AllowCURPutObject",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${data.aws_billing_service_account.test.arn}"
      },
      "Action": "s3:PutObject",
      "Resource": "arn:${data.aws_partition.current.partition}:s3:::${aws_s3_bucket.test.id}/*"
    }
  ]
}
POLICY
}

resource "aws_cur_report_definition" "test" {
  depends_on = [aws_s3_bucket_policy.test] # needed to avoid "ValidationException: Failed to verify customer bucket permission."

  report_name                = %[1]q
  time_unit                  = "DAILY"
  format                     = "textORcsv"
  compression                = "GZIP"
  additional_schema_elements = ["RESOURCES", "SPLIT_COST_ALLOCATION_DATA"]
  s3_bucket                  = aws_s3_bucket.test.id
  s3_prefix                  = ""
  s3_region                  = aws_s3_bucket.test.region
  additional_artifacts       = ["REDSHIFT", "QUICKSIGHT"]
  refresh_closed_reports     = true
  report_versioning          = "CREATE_NEW_REPORT"
}

data "aws_cur_report_definition" "test" {
  report_name = aws_cur_report_definition.test.report_name
}
`, reportName, bucketName)
}
