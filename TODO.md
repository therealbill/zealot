TODO
 * Integrate workspaces?
 * Build working directory from name and resource?
 * Or use /tmp/+Consul value ?
 * Main thing is the absolute path must be the
   same. If run in Nomad, it will be rooted/contained so there should be no risk
   of leakage.
 * Look at using Afero to do it all in memory - the problem is go-getter
   downloading the version to local filesystem, though that may be an
   acceptable exception. The main question is whether I can execute between the two.
 * Refactor Terraform Module struct to an interface to support multiple resource
  types in a single binary?
 * Move resource definitions to their own file(s)
