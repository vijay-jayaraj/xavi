## XAVI - The XTRAC API Platform

Xavi is a software layer that decouples API consumers from the systems that provide the 
underlying capabilities of the APIs, and allows an API contract to be defined and maintained from 
the consumer perspective, implemented as a facade in front of the services and applications providing 
API functionality.

### Contributing

To contribute, you must certify you agree with the [Developer Certificate of Origin](http://developercertificate.org/)
by signing your commits via `git -s`. To create a signature, configure your user name and email address in git.
Sign with your real name, do not use pseudonyms or submit anonymous commits.


In terms of workflow:

0. For significant changes or improvement, create an issue before commencing work.
1. Fork the respository, and create a branch for your edits.
2. Add tests that cover your changes, unit tests for smaller changes, acceptance test
for more significant functionality.
3. Run gofmt on each file you change before committing your changes.
4. Run golint on each file you change before committing your changes.
5. Make sure all the tests pass before committing your changes.
6. Commit your changes and issue a pull request.

### License

(c) 2015 Fidelity Investments
Licensed under the Apache License, Version 2.0


### Dependency management

Dependencies are managed via [Godep](https://github.com/tools/godep) using golang 1.5 vendoring support.
Refer to the documentation for how
to manage dependencies, how to preface go commands with godep to pick up stored dependencies, etc.

[This article](http://www.goinggo.net/2013/10/manage-dependencies-with-godep.html) provides a nice overview.

Use godep to build, clean, and test, e.g.

<pre>
godep go build
godep go test $(go list ./... | grep -v /vendor/)
godep go clean
</pre>

Note how ./... can't be used as normal due to it recursing into the vendor directory.

#### Crypto Package

The cryto package has now been vendored via Godeps. The proper path for go get
and vendoring is golang.org/x/crypto/ssh

There are still some hassles with the crypto package, for example a godep restore
fails, complaining about the crypto ssh terminal import path, e.g.

<pre>
godep: unrecognized import path "golang.org/x/crypto/ssh/terminal"
</pre>

### Codeship Build Setup

Setup Commands

<pre>
cd $HOME
wget https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz
tar xvzf go1.5.1.linux-amd64.tar.gz
export GOROOT=$HOME/go
export PATH=$GOROOT/bin:$PATH
go get github.com/tools/godep
export GO15VENDOREXPERIMENT=1
go version
cd $GOPATH/src/github.com/xtracdev/xavi
</pre>

Test Commands

<pre>
godep go test $(go list ./... | grep -v /vendor/)
</pre>


### KVStore

XAVI can work with consul as the configuration store, or use a hashmap-based KVStore that can dump and load its
contents to/from a file.

Use the XAVI_KVSTORE_URL environment variable to set the KVStore URL. For consul, the value of the environment
variable should be `consul://host:port` and for a the hashmap/file store specify a file URL for the hashmap backing
file. Note that file URLs are full paths to files, e.g.

<pre>
export XAVI_KVSTORE_URL=file:////some/path/democfg.xavi
</pre>



### Logging

[Logrus](https://github.com/Sirupsen/logrus) is currently the logging framework. The log level can be set
via the XAVI_LOGGING_LEVEL environment variable (valid values are debug, info, warn, fatal, error, panic).



### Cross-compiling

For details on cross compiling see https://gist.github.com/d-smith/9d7ca1baa72135dfe7b0

TL;DR

<pre>
GOOS=linux GOARCH=386 CGO_ENABLED=0 godep go build
</pre>



### Go Code Coverage

		godep go test -coverprofile=coverage.out; go tool cover -html=coverage.out
		
#### Go Code Coverage with Gocov

Gocov will accumulate coverage recursively, unlike the go test tool which produces coverage for a single
package.

gocov test $(go list ./... | grep -v /vendor/) |gocov-html > coverage.html

### Port Usage - Mac Os X  

	lsof -n -i4TCP


### Demo Service With Mountebank

We can use [Mountebank](http://www.mbtest.org/) as a service endpoint for trying out XAVI.

Consider the following mountebank imposter definition (see democonfig/imposter.json):

<pre>
{
  "port": 4545,
  "protocol": "http",
  "stubs": [
    {
      "responses": [
        {
          "is": {
            "statusCode": 400,
            "body": "All work and no play makes Jack a dull boy.\nAll work and no play makes Jack a dull boy.\nAll work and no play makes Jack a dull boy.\nAll work and no play makes Jack a dull boy.\n"
          }
        }
      ],
      "predicates": [
        {
          "equals": {
            "path": "/hello",
            "method": "GET"
          }
        }
      ]
    }
  ]
}
</pre>

We can provide a mock /hello service by defining it using Mountebank:

<pre>
curl -i -X POST -H 'Content-Type: application/json' -d@democonfig/imposter.json http://127.0.0.1:2525/imposters
</pre>

The service endpoint can be called via curl:

<pre>
curl localhost:4545/hello
All work and no play makes Jack a dull boy.
All work and no play makes Jack a dull boy.
All work and no play makes Jack a dull boy.
All work and no play makes Jack a dull boy.
</pre>

We can then set up a simple proxy example like this:

<pre>
export XAVI_KVSTORE_URL=file:////some/path/mbdemo.xavi
curl -i -X POST -H 'Content-Type: application/json' -d@democonfig/imposter.json http://127.0.0.1:2525/imposters
./xavi add-server -address localhost -port 4545 -name hello1 -ping-uri /hello
./xavi add-backend -name demo-backend -servers hello1
./xavi add-route -name demo-route -backend demo-backend -base-uri /hello
./xavi add-listener -name demo-listener -routes demo-route
./xavi listen -ln demo-listener -address 0.0.0.0:8080
</pre>

After the listener is started, the proxy endpoint can be used, e.g. `curl localhost:8080/hello`

For a two server round-robin proxy config demo, try this:

<pre>
export XAVI_KVSTORE_URL=file:////some/path/mbdemo.xavi
curl -i -X POST -H 'Content-Type: application/json' -d@democonfig/hello3000.json http://127.0.0.1:2525/imposters
curl -i -X POST -H 'Content-Type: application/json' -d@democonfig/hello3100.json http://127.0.0.1:2525/imposters
./xavi add-server -address localhost -port 3000 -name hello1 -ping-uri /hello -health-check http-get
./xavi add-server -address localhost -port 3100 -name hello2 -ping-uri /hello -health-check http-get
./xavi add-backend -name demo-backend -servers hello1,hello2 -load-balancer-policy round-robin
./xavi add-route -name demo-route -backend demo-backend -base-uri /hello
./xavi add-listener -name demo-listener -routes demo-route
./xavi listen -ln demo-listener -address 0.0.0.0:8080
</pre>


Curling the endpoint as in the single server instance, the mountebank log shows the requests
being distributed to the two endpoints.

<pre>
info: [http:3000] ::ffff:127.0.0.1:50568 => GET /hello
info: [http:3100] ::ffff:127.0.0.1:50571 => GET /hello
info: [http:3000] ::ffff:127.0.0.1:50573 => GET /hello
info: [http:3100] ::ffff:127.0.0.1:50575 => GET /hello
</pre>



### Acceptance Testing Setup with Docker-Machine

Assuming you are working on a mac, you'll need to install the docker tools and docker machine
to run docker, which is needed for running the Xavi acceptance tests.

If you are attached to a network that uses an http proxy to connect to the internet, you'll need
to update the docker daemon proxy settings in the docker VM. To do so:

1. Connect to the default machine via `docker-machine -D ssh default`
2. `sudo vi /var/lib/boot2docker/profile ` and export proxy setting environment variables
(export HTTP_PROXY=http://<proxy host>:<proxy port>, export HTTPS_PROXY=http://<proxy host>:<proxy port>
placed on separate lines).
3. Restart the VM. You can use the Virtual Box client to do this.
4. You will also need to edit the Dockerfiles mentioned below to uncomment out the proxy ENV
lines, and add your proxy server IP address and port.

The tests are written assuming the following port configuration:

<pre>
VBoxManage controlvm default natpf1 "standalone-mb,tcp,127.0.0.1,3636,,2626"
VBoxManage controlvm default natpf1 "cohosted-mb,tcp,127.0.0.1,3535,,2525"
VBoxManage controlvm default natpf1 "xavi-rest-agent,tcp,127.0.0.1,8080,,8080"
VBoxManage controlvm default natpf1 "xavi-test-server,tcp,127.0.0.1,9000,,9000"
</pre>

#### Mountebank

Note the above port forwarding - it maps the host os perspective to the guest os
perspective, with docker port forwarding managing the mapping of the guest os to the
container. 

Also note that there is no need to start the containers using docker - the acceptance tests will
check to see if the container is running. If the container is not running, the acceptance test will
create and start it using the docker API. The commands below are provided as documentation of what the
acceptance test code does to run the containers.

Build the Mountebank server image in the docker/mountebank-alpine directory:

<pre>
docker build -t "mb-server-alpine" .
docker run -d -p 2626:2525 --name mountebank --label 'xt-container-type=atest-mb' mb-server-alpine
</pre>

#### XAVI Docker set up

Note that you must first cross-compile xavi for linux and copy it into docker/xavi-alpine 
before building the image, e.g. `GOOS=linux GOARCH=386 CGO_ENABLED=0 godep go build`

Build the image in the docker/xavi-alpine directory

<pre>
docker build -t "xavi-test-alpine-base" .
docker run -d -p 8080:8080 -p 9000:9000 -p 2525:2525 --name xavi-docker --label 'xt-container-type=atest-xavi' --link mountebank:mbhost xavi-test-alpine-base
</pre>


By default the containers are stopped at the end of each test run, which is best for ensure clean test environments in
a known state, as opposed to containers in indeterminate states, for example if test failures leaves ports tied up,
test configuration that can't be cleaned up, etc.

For the purpose of quickly running tests that are meant to test the introduction of breaking changes, the containers
can be left running if gucumber is executued with the environment variable XT_CLEANUP_CONTAINERS=false, e.g.

<pre>
env XT_CLEANUP_CONTAINERS=false gucumber
</pre>




