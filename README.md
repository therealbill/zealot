# Zealot

A tool for defining Terraform resources in consul and applying them in a stateless job.

# What it does
 * Fetch module template from Consul 
 * Fetch Variables from Consul 
 * If needed, secrets are expected to be handled by Nomad
 * Generate terraform file combining template and variables 
 * Run terraform init && apply 
 * Upload plan and plan text to Consul 

If autoapply is true AND there are changes in the plan:
* Run Terraform apply
Else:
* skip apply

Zealot uses Consul for remote state storage. This is done to handle concurrency
as well as to maintain the native terraform plan/apply state checking.

# Terraform 

Zealot will retrieve the version of terraform desired autoamtically. Currently
this is hardcoded to "0.11.1", but I want to move that into a default version
under `appconfig/zealot/` and the ability to override in the jobconfig. This
would let each resource/job specify a version for such cases as testing a new
TF version, requiring a newer one for features, or even using an old one for
compatability. Of course, once a job is created, you can only go up in
versions, since state includes the version it needs.

# Expected Consul KV structure

Zealot expects three "root" URL paths in Consul's KV store.
```
appconfig/
jobconfig/
states/
```

## appconfig

This is where Zealot will look for templates and any Zealot specific config.
For example, a `local_file` resource would store its template content in
`appconfig/zealot/local_file/template`.

A given resource could use the `local_file/` tree to add "default" parameters
if the code was adapted to consult it.

## jobconfig

This is where Zealot will look for the resource it needs to manage via
terraform. For example, if we have a `local_file` resource named "myfile", then
the root config for that "job" would be `jobconfig/zealot/myfile`.

### local_file
_This is an example_

The `local_file` resource expects the following Consul KV structure.

```
autoapply
PlanText
ChangesAvailable
planfile
WorkingDir
module/Filename
module/Content
module/ResourceName
```

The 'autoapply' and 'ChangesAvailable' values must be true or false.  PlanText will
contain the output from the most recent Zealot plan, while 'planfile' contains
the actual plan from it. WorkingDir is where the files and work will be done.

If anything other than Content in `module` are empty, no resource will exist in
the generated Terraform file, causing terraform to want to delete any existign
resource.

## Job States

The module subtree defines the filename, full path if not local, the resource
name to be used in terraform, and the content of the file.

Since this is intended for "ephemeral" infrastructure such as development
resources the state of a given job resource is kept in
`jobconfig/zealot/NAME/state` rather than the root states tree. This allows you
to use ACLs to control state access mre broadly while allowing the job to
manage its own state.

If you wanted to have a UI for managing the jobs you would be able to get
everythign you need to know about the job from the job's root.


# Event Handling
I'd like to integrate the ability to send/receive events to trigger outside
things. One route that would be easy to send to without additional setup is to
use Consul events. I'm also considering AWS SQS/SNS and NSQ.

Another way to go would be to add the ability to call a webhook. This would be
configured at the Zealot config level and would pass a JSON of the important
bits such as the job that needs updating. By doing this, Zealot itself could,
if a plan had changes, call a webhook that sent an email, schedule a review,
mention it in a slack room, and so on.

# Watches

One could alter Zealot to run as a "service" where it executed a Consul watch
on the resource it manages, thus picking up changes automatically. This would
require an instance per resource, but could be more responsive.

# Applying Changes

Whether a change will be *applied* by Zealot is determined by the autoapply flag in the job's config.
