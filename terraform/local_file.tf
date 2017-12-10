provider "consul" {
	address = "localhost:8500"
}

terraform {
    backend "consul" {
        address = "localhost:8500"
        path = "states/zealot/local_file/state"
    }
}

data "template_file" "local_file_template" {
  template = "${file("templates/local_file_template")}"
}

resource "consul_key_prefix" "zealot_template" {
	path_prefix = "appconfig/zealot/"
	subkeys = {
		"local_file/template" = "${data.template_file.local_file_template.rendered}"
	}
}
