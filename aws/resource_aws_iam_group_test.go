package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestValidateIamGroupName(t *testing.T) {
	validNames := []string{
		"test-group",
		"test_group",
		"testgroup123",
		"TestGroup",
		"Test-Group",
		"test.group",
		"test.123,group",
		"testgroup@hashicorp",
		"test+group@hashicorp.com",
	}
	for _, v := range validNames {
		_, errs := validateAwsIamGroupName(v, "name")
		if len(errs) != 0 {
			t.Fatalf("%q should be a valid IAM Group name: %q", v, errs)
		}
	}

	invalidNames := []string{
		"!",
		"/",
		" ",
		":",
		";",
		"test name",
		"/slash-at-the-beginning",
		"slash-at-the-end/",
	}
	for _, v := range invalidNames {
		_, errs := validateAwsIamGroupName(v, "name")
		if len(errs) == 0 {
			t.Fatalf("%q should be an invalid IAM Group name", v)
		}
	}
}

func TestAccAWSIAMGroup_importBasic(t *testing.T) {
	resourceName := "aws_iam_group.group"

	rString := acctest.RandString(8)
	groupName := fmt.Sprintf("tf-acc-group-import-%s", rString)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGroupConfig(groupName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSIAMGroup_basic(t *testing.T) {
	var conf iam.GetGroupOutput

	rString := acctest.RandString(8)
	groupName := fmt.Sprintf("tf-acc-group-basic-%s", rString)
	groupName2 := fmt.Sprintf("tf-acc-group-basic-2-%s", rString)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGroupConfig(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupExists("aws_iam_group.group", &conf),
					testAccCheckAWSGroupAttributes(&conf, groupName, "/"),
				),
			},
			{
				Config: testAccAWSGroupConfig2(groupName2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupExists("aws_iam_group.group2", &conf),
					testAccCheckAWSGroupAttributes(&conf, groupName2, "/funnypath/"),
				),
			},
		},
	})
}

func TestAccAWSIAMGroup_nameChange(t *testing.T) {
	var conf iam.GetGroupOutput

	rString := acctest.RandString(8)
	groupName := fmt.Sprintf("tf-acc-group-basic-%s", rString)
	groupName2 := fmt.Sprintf("tf-acc-group-basic-2-%s", rString)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGroupConfig(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupExists("aws_iam_group.group", &conf),
					testAccCheckAWSGroupAttributes(&conf, groupName, "/"),
				),
			},
			{
				Config: testAccAWSGroupConfig(groupName2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGroupExists("aws_iam_group.group", &conf),
					testAccCheckAWSGroupAttributes(&conf, groupName2, "/"),
				),
			},
		},
	})
}

func testAccCheckAWSGroupDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_group" {
			continue
		}

		// Try to get group
		_, err := iamconn.GetGroup(&iam.GetGroupInput{
			GroupName: aws.String(rs.Primary.ID),
		})
		if err == nil {
			return errors.New("still exist.")
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "NoSuchEntity" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSGroupExists(n string, res *iam.GetGroupOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Group name is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn

		resp, err := iamconn.GetGroup(&iam.GetGroupInput{
			GroupName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}

		*res = *resp

		return nil
	}
}

func testAccCheckAWSGroupAttributes(group *iam.GetGroupOutput, name string, path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.Group.GroupName != name {
			return fmt.Errorf("Bad name: %s when %s was expected", *group.Group.GroupName, name)
		}

		if *group.Group.Path != path {
			return fmt.Errorf("Bad path: %s when %s was expected", *group.Group.Path, path)
		}

		return nil
	}
}

func testAccAWSGroupConfig(groupName string) string {
	return fmt.Sprintf(`
resource "aws_iam_group" "group" {
  name = "%s"
  path = "/"
}
`, groupName)
}

func testAccAWSGroupConfig2(groupName string) string {
	return fmt.Sprintf(`
resource "aws_iam_group" "group2" {
  name = "%s"
  path = "/funnypath/"
}
`, groupName)
}
