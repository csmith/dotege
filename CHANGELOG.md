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
