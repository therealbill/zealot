# Terraforming Consul

NOTE: I'll probably move these around a bit. The main question is whether wI
want a state for Zealot or one for each resource type. I lean heavily toward
Zealot to keep it simple and concise.

Basically this tree is for terraform files to upload the template content for
each defined resource type. If any default variables also need to set, do that
in the resource's TF file.  Also, use `consul_keys_prefix` to ensure complete
control over that prefix.

The use of a template file for the content allows you do write the template
file normally without needing to worry about escaping it.
