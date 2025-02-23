package apigateway

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_api_gateway_domain_name", name="Domain Name")
// @Tags(identifierAttribute="arn")
func ResourceDomainName() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceDomainNameCreate,
		ReadWithoutTimeout:   resourceDomainNameRead,
		UpdateWithoutTimeout: resourceDomainNameUpdate,
		DeleteWithoutTimeout: resourceDomainNameDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			//According to AWS Documentation, ACM will be the only way to add certificates
			//to ApiGateway DomainNames. When this happens, we will be deprecating all certificate methods
			//except certificate_arn. We are not quite sure when this will happen.
			"certificate_arn": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"certificate_body", "certificate_chain", "certificate_name", "certificate_private_key", "regional_certificate_arn", "regional_certificate_name"},
			},
			"certificate_body": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"certificate_arn", "regional_certificate_arn"},
			},
			"certificate_chain": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"certificate_arn", "regional_certificate_arn"},
			},
			"certificate_name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"certificate_arn", "regional_certificate_arn", "regional_certificate_name"},
			},
			"certificate_private_key": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"certificate_arn", "regional_certificate_arn"},
			},
			"certificate_upload_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cloudfront_domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cloudfront_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"endpoint_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"types": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							// BadRequestException: Cannot create an api with multiple Endpoint Types
							MaxItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									apigateway.EndpointTypeEdge,
									apigateway.EndpointTypeRegional,
								}, false),
							},
						},
					},
				},
			},
			"mutual_tls_authentication": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"truststore_uri": {
							Type:     schema.TypeString,
							Required: true,
						},
						"truststore_version": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"ownership_verification_certificate_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: verify.ValidARN,
			},
			"regional_certificate_arn": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"certificate_arn", "certificate_body", "certificate_chain", "certificate_name", "certificate_private_key", "regional_certificate_name"},
			},
			"regional_certificate_name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"certificate_arn", "certificate_name", "regional_certificate_arn"},
			},
			"regional_domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"regional_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_policy": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(apigateway.SecurityPolicy_Values(), true),
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceDomainNameCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).APIGatewayConn(ctx)

	domainName := d.Get("domain_name").(string)
	input := &apigateway.CreateDomainNameInput{
		DomainName:              aws.String(domainName),
		MutualTlsAuthentication: expandMutualTLSAuthentication(d.Get("mutual_tls_authentication").([]interface{})),
		Tags:                    getTagsIn(ctx),
	}

	if v, ok := d.GetOk("certificate_arn"); ok {
		input.CertificateArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("certificate_body"); ok {
		input.CertificateBody = aws.String(v.(string))
	}

	if v, ok := d.GetOk("certificate_chain"); ok {
		input.CertificateChain = aws.String(v.(string))
	}

	if v, ok := d.GetOk("certificate_name"); ok {
		input.CertificateName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("certificate_private_key"); ok {
		input.CertificatePrivateKey = aws.String(v.(string))
	}

	if v, ok := d.GetOk("endpoint_configuration"); ok {
		input.EndpointConfiguration = expandEndpointConfiguration(v.([]interface{}))
	}

	if v, ok := d.GetOk("ownership_verification_certificate_arn"); ok {
		input.OwnershipVerificationCertificateArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("regional_certificate_arn"); ok {
		input.RegionalCertificateArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("regional_certificate_name"); ok {
		input.RegionalCertificateName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_policy"); ok {
		input.SecurityPolicy = aws.String(v.(string))
	}

	output, err := conn.CreateDomainNameWithContext(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating API Gateway Domain Name (%s): %s", domainName, err)
	}

	d.SetId(aws.StringValue(output.DomainName))

	return append(diags, resourceDomainNameRead(ctx, d, meta)...)
}

func resourceDomainNameRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).APIGatewayConn(ctx)

	domainName, err := FindDomainName(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] API Gateway Domain Name (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading API Gateway Domain Name (%s): %s", d.Id(), err)
	}

	arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   "apigateway",
		Region:    meta.(*conns.AWSClient).Region,
		Resource:  fmt.Sprintf("/domainnames/%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	d.Set("certificate_arn", domainName.CertificateArn)
	d.Set("certificate_name", domainName.CertificateName)
	if domainName.CertificateUploadDate != nil {
		d.Set("certificate_upload_date", domainName.CertificateUploadDate.Format(time.RFC3339))
	} else {
		d.Set("certificate_upload_date", nil)
	}
	d.Set("cloudfront_domain_name", domainName.DistributionDomainName)
	d.Set("cloudfront_zone_id", meta.(*conns.AWSClient).CloudFrontDistributionHostedZoneID())
	d.Set("domain_name", domainName.DomainName)
	if err := d.Set("endpoint_configuration", flattenEndpointConfiguration(domainName.EndpointConfiguration)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting endpoint_configuration: %s", err)
	}
	if err = d.Set("mutual_tls_authentication", flattenMutualTLSAuthentication(domainName.MutualTlsAuthentication)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting mutual_tls_authentication: %s", err)
	}
	d.Set("ownership_verification_certificate_arn", domainName.OwnershipVerificationCertificateArn)
	d.Set("regional_certificate_arn", domainName.RegionalCertificateArn)
	d.Set("regional_certificate_name", domainName.RegionalCertificateName)
	d.Set("regional_domain_name", domainName.RegionalDomainName)
	d.Set("regional_zone_id", domainName.RegionalHostedZoneId)
	d.Set("security_policy", domainName.SecurityPolicy)

	setTagsOut(ctx, domainName.Tags)

	return diags
}

func resourceDomainNameUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).APIGatewayConn(ctx)

	if d.HasChangesExcept("tags", "tags_all") {
		var operations []*apigateway.PatchOperation

		if d.HasChange("certificate_arn") {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/certificateArn"),
				Value: aws.String(d.Get("certificate_arn").(string)),
			})
		}

		if d.HasChange("certificate_name") {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/certificateName"),
				Value: aws.String(d.Get("certificate_name").(string)),
			})
		}

		if d.HasChange("endpoint_configuration.0.types") {
			// The domain name must have an endpoint type.
			// If attempting to remove the configuration, do nothing.
			if v, ok := d.GetOk("endpoint_configuration"); ok && len(v.([]interface{})) > 0 {
				m := v.([]interface{})[0].(map[string]interface{})

				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/endpointConfiguration/types/0"),
					Value: aws.String(m["types"].([]interface{})[0].(string)),
				})
			}
		}

		if d.HasChange("mutual_tls_authentication") {
			if v, ok := d.GetOk("mutual_tls_authentication"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
				tfMap := v.([]interface{})[0].(map[string]interface{})

				if d.HasChange("mutual_tls_authentication.0.truststore_uri") {
					operations = append(operations, &apigateway.PatchOperation{
						Op:    aws.String(apigateway.OpReplace),
						Path:  aws.String("/mutualTlsAuthentication/truststoreUri"),
						Value: aws.String(tfMap["truststore_uri"].(string)),
					})
				}

				if d.HasChange("mutual_tls_authentication.0.truststore_version") {
					operations = append(operations, &apigateway.PatchOperation{
						Op:    aws.String(apigateway.OpReplace),
						Path:  aws.String("/mutualTlsAuthentication/truststoreVersion"),
						Value: aws.String(tfMap["truststore_version"].(string)),
					})
				}
			} else {
				// To disable mutual TLS for a custom domain name, remove the truststore from your custom domain name.
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/mutualTlsAuthentication/truststoreUri"),
					Value: aws.String(""),
				})
			}
		}

		if d.HasChange("regional_certificate_arn") {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/regionalCertificateArn"),
				Value: aws.String(d.Get("regional_certificate_arn").(string)),
			})
		}

		if d.HasChange("regional_certificate_name") {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/regionalCertificateName"),
				Value: aws.String(d.Get("regional_certificate_name").(string)),
			})
		}

		if d.HasChange("security_policy") {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/securityPolicy"),
				Value: aws.String(d.Get("security_policy").(string)),
			})
		}

		_, err := conn.UpdateDomainNameWithContext(ctx, &apigateway.UpdateDomainNameInput{
			DomainName:      aws.String(d.Id()),
			PatchOperations: operations,
		})

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating API Gateway Domain Name (%s): %s", d.Id(), err)
		}

		if _, err := waitDomainNameUpdated(ctx, conn, d.Id()); err != nil {
			return sdkdiag.AppendErrorf(diags, "waiting for API Gateway Domain Name (%s) update: %s", d.Id(), err)
		}
	}

	return append(diags, resourceDomainNameRead(ctx, d, meta)...)
}

func resourceDomainNameDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).APIGatewayConn(ctx)

	log.Printf("[DEBUG] Deleting API Gateway Domain Name: %s", d.Id())
	_, err := conn.DeleteDomainNameWithContext(ctx, &apigateway.DeleteDomainNameInput{
		DomainName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, apigateway.ErrCodeNotFoundException) {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting API Gateway Domain Name (%s): %s", d.Id(), err)
	}

	return diags
}

func FindDomainName(ctx context.Context, conn *apigateway.APIGateway, domainName string) (*apigateway.DomainName, error) {
	input := &apigateway.GetDomainNameInput{
		DomainName: aws.String(domainName),
	}

	output, err := conn.GetDomainNameWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, apigateway.ErrCodeNotFoundException) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output, nil
}

func statusDomainName(ctx context.Context, conn *apigateway.APIGateway, domainName string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindDomainName(ctx, conn, domainName)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}
		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.DomainNameStatus), nil
	}
}

func waitDomainNameUpdated(ctx context.Context, conn *apigateway.APIGateway, domainName string) (*apigateway.DomainName, error) {
	const (
		timeout = 15 * time.Minute
	)
	stateConf := &retry.StateChangeConf{
		Pending:    []string{apigateway.DomainNameStatusUpdating},
		Target:     []string{apigateway.DomainNameStatusAvailable},
		Refresh:    statusDomainName(ctx, conn, domainName),
		Timeout:    timeout,
		Delay:      1 * time.Minute,
		MinTimeout: 10 * time.Second,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*apigateway.DomainName); ok {
		tfresource.SetLastError(err, errors.New(aws.StringValue(output.DomainNameStatusMessage)))

		return output, err
	}

	return nil, err
}

func expandMutualTLSAuthentication(tfList []interface{}) *apigateway.MutualTlsAuthenticationInput {
	if len(tfList) == 0 || tfList[0] == nil {
		return nil
	}

	tfMap := tfList[0].(map[string]interface{})

	apiObject := &apigateway.MutualTlsAuthenticationInput{}

	if v, ok := tfMap["truststore_uri"].(string); ok && v != "" {
		apiObject.TruststoreUri = aws.String(v)
	}

	if v, ok := tfMap["truststore_version"].(string); ok && v != "" {
		apiObject.TruststoreVersion = aws.String(v)
	}

	return apiObject
}

func flattenMutualTLSAuthentication(apiObject *apigateway.MutualTlsAuthentication) []interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.TruststoreUri; v != nil {
		tfMap["truststore_uri"] = aws.StringValue(v)
	}

	if v := apiObject.TruststoreVersion; v != nil {
		tfMap["truststore_version"] = aws.StringValue(v)
	}

	return []interface{}{tfMap}
}
