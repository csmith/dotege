# v2.0.0 (unreleased)

## Major breaking changes

* Remove support for SSL certificate generation and deployment. I'm not aware
  of any deployment of Dotege actually using this functionality, and it's a
  non-trivial maintenance burden. If you _are_ using this functionality, please
  get in touch (and stick with v1 for now!)
* Remove support for ACLs and users. This was very difficult to configure, and
  is an extremely niche use case. This will be replaced with a more general way
  to pass custom data to templates, which can then use it how they see fit.

## Other changes

* Added a `DOTEGE_POLL` option that makes Dotege poll the container list
  instead of using events. This may help to mitigate some issues caused by
  strange state when using docker-compose.
* The `DOTEGE_DEBUG` env var is no longer used. Dotege's default logging will
  be slightly more verbose.
* The format of Dotege's logs have changed. They are now formatted using Go's
  standard logger instead of a third-party logging tool.

# v1.3.2

## Other changes

* Update golang.org/x/net dependency to fix building on newer Go versions
* Update to Go 1.23

# v1.3.1

## Bug fixes

* Domain names in certificates and templates are now ordered consistently,
  in the order they're specified in the `com.chameth.vhost` label. Previously,
  these were accidentally alphabetised in a lot of situations.

# v1.3.0

## Features

* Dotege can now deploy private keys separately to their corresponding
  certificates by setting `DOTEGE_CERTIFICATE_DEPLOYMENT` to `splitkeys`.
  (Thanks @Greboid)

## Other changes

* Update to Go 1.18
* Miscellaneous dependency updates

# v1.2.0

## Features

* Dotege can now be configured to not manage TLS certificates at all.
  When `DOTEGE_CERTIFICATE_DEPLOYMENT` is set to `disabled` no certificates
  will be requested or written to disk, and all certificate-related options
  are ignored.

## Other changes

* Updated the default haproxy template (thanks @Greboid):
  * Updated cipher suites in line with Mozilla's current intermediate recommendations
  * Don't overwrite the Strict-Transport-Security header if sent by upstream
  * Remove any Server header returned from upstream
* Miscellaneous dependency updates

# v1.1.0

## Features

* You can now limit what containers Dotege will monitor by specifying the
  `DOTEGE_PROXYTAG` env var. Only containers with a matching `com.chameth.proxytag`
  label will then be used when generating templates.
* You can now use build tags when compiling Dotege to restrict it to a single
  DNS providers for ACME authentications. For example building with
  `-tags lego_httpreq` only includes HTTPREQ and shaves around 30MB from the
  resulting binary.

## Other changes

* Update to Go 1.17
* Miscellaneous dependency updates
