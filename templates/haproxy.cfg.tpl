global
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options prefer-client-ciphers no-sslv3 no-tlsv10 no-tlsv11 no-tls-tickets
    ssl-default-server-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
    ssl-default-server-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-server-options no-sslv3 no-tlsv10 no-tlsv11 no-tls-tickets

resolvers docker_resolver
    nameserver dns 127.0.0.11:53

defaults
    log global
    mode    http
    timeout connect 5000
    timeout client 30000
    timeout server 30000
    compression algo gzip
    compression type text/plain text/css application/json application/javascript application/x-javascript text/xml application/xml application/xml+rss text/javascript
    default-server init-addr last,libc,none check resolvers docker_resolver

frontend main
    mode    http
    bind    :::443 v4v6 ssl strict-sni alpn h2,http/1.1 crt /certs/
    bind    :::80 v4v6
    http-request set-header X-Forwarded-For %[src]
    http-request set-header X-Forwarded-Proto https if { ssl_fc }
    redirect scheme https code 301 if !{ ssl_fc }
    http-response set-header Strict-Transport-Security max-age=15768000 if { res.fhdr_cnt(Strict-Transport-Security) 0 }
    http-response del-header Server
{{- range .Hostnames }}
    use_backend {{ .Name | replace "." "_" }} if { hdr(host) -i {{ .Name }}
        {{- range .Alternatives }} || hdr(host) -i {{ . }} {{- end }} }
{{- end -}}

{{ range .Hostnames }}

backend {{ .Name | replace "." "_" }}
    mode http
    {{- range .Containers }}
        {{- if .ShouldProxy }}
    server server1 {{ .Name }}:{{ .Port }}
        {{- end -}}
    {{- end -}}
    {{- range $k, $v := .Headers }}
    http-response set-header {{ $k }} "{{ $v | replace "\"" "\\\"" }}"
    {{- end -}}
{{ end }}
