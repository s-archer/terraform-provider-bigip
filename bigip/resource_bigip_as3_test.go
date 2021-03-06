/*
Copyright 2019 F5 Networks Inc.
This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
If a copy of the MPL was not distributed with this file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/
package bigip

import (
	"crypto/tls"
	"fmt"
	"github.com/f5devcentral/go-bigip"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"testing"
)

//var TEST_DEVICE_NAME = fmt.Sprintf("/%s/test-device", TEST_PARTITION)

var dir, err = os.Getwd()

var TestAs3Resource = `
resource "bigip_as3"  "as3-example" {
     as3_json = "${file("` + dir + `/../examples/as3/example1.json")}"
}
`
var TestAs3Resource1 = `
resource "bigip_as3"  "as3-multitenant-example" {
     as3_json = "${file("` + dir + `/../examples/as3/as3_example1.json")}"
}
`
var TestAs3Resource2 = `
resource "bigip_as3"  "as3-partialsuccess-example" {
     as3_json = "${file("` + dir + `/../examples/as3/as3_example2.json")}"
}
`
var TestAs3Resource3 = `
resource "bigip_as3"  "as3-tenantadd-example" {
     as3_json = "${file("` + dir + `/../examples/as3/as3_example3.json")}"
}
`
var TestAs3Resource4 = `
resource "bigip_as3"  "as3-tenantfilter-example" {
     as3_json = "${file("` + dir + `/../examples/as3/as3_example1.json")}"
     tenant_filter = "Sample_01"
}
`
var TestAs3ResourceInvalidJson = `
resource "bigip_as3"  "as3-example" {
     as3_json = "${file("` + dir + `/../examples/as3/invalid.json")}"
}
`

func TestAccBigipAs3_create_SingleTenant(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckAs3Destroy,
		Steps: []resource.TestStep{
			{
				Config: TestAs3Resource,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_new", true),
				),
			},
		},
	})
}

func TestAccBigipAs3_create_MultiTenants(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckAs3Destroy,
		Steps: []resource.TestStep{
			{
				Config: TestAs3Resource1,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_01,Sample_02", true),
				),
			},
		},
	})
}
func TestAccBigipAs3_create_PartialSuccess(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckAs3Destroy,
		Steps: []resource.TestStep{
			{
				Config: TestAs3Resource2,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_03", true),
					testCheckAs3Exists("Sample_04", false),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
func TestAccBigipAs3_addTenantFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckAs3Destroy,
		Steps: []resource.TestStep{
			{
				Config: TestAs3Resource4,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_01", true),
					testCheckAs3Exists("Sample_02", false),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccBigipAs3_update_addTenant(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckAs3Destroy,
		Steps: []resource.TestStep{
			{
				Config: TestAs3Resource1,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_01,Sample_02", true),
				),
			},
			{
				Config: TestAs3Resource3,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_01,Sample_02,Sample_03", true),
				),
			},
		},
	})
}
func TestAccBigipAs3_update_deleteTenant(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckAs3Destroy,
		Steps: []resource.TestStep{
			{
				Config: TestAs3Resource3,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_01,Sample_02,Sample_03", true),
				),
			},
			{
				Config: TestAs3Resource1,
				Check: resource.ComposeTestCheckFunc(
					testCheckAs3Exists("Sample_01,Sample_02", true),
					testCheckAs3Exists("Sample_03", false),
				),
			},
		},
	})
}

func testCheckAs3Exists(name string, exists bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientBigip := testAccProvider.Meta().(*bigip.BigIP)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		client := &http.Client{Transport: tr}
		url := clientBigip.Host + "/mgmt/shared/appsvcs/declare/" + name
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("[ERROR] Error while creating http request with AS3 json: %v", err)
		}
		req.SetBasicAuth(clientBigip.User, clientBigip.Password)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		body, err := ioutil.ReadAll(resp.Body)
		bodyString := string(body)
		if (resp.Status == "204 No Content" || err != nil || resp.StatusCode == 404) && exists {
			return fmt.Errorf("[ERROR] Error while checking as3resource present in bigip :%s  %v", bodyString, err)
			defer resp.Body.Close()
		}
		defer resp.Body.Close()
		return nil
	}
}

func TestAccBigipAs3_badJSON(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAcctPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckdevicesDestroyed,
		Steps: []resource.TestStep{
			{
				Config:      TestAs3ResourceInvalidJson,
				ExpectError: regexp.MustCompile(`"as3_json" contains an invalid JSON:.*`),
			},
		},
	})
}
func testCheckAs3Destroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*bigip.BigIP)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bigip_as3" {
			continue
		}

		name := rs.Primary.ID
		err, failedTenants := client.DeleteAs3Bigip(name)
		if err != nil || failedTenants != "" {
			return err
		}
	}
	return nil
}
