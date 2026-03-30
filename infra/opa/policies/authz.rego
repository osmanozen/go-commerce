package ecommerce.authz

import rego.v1

default allow := false

# Example policy:
# - GET requests are allowed
# - non-GET requests require role=admin
allow if {
	input.method == "GET"
}

allow if {
	input.method != "GET"
	input.role == "admin"
}
