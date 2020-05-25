; zone file fragment for {{.Name}}

$TTL {{.TTL}}

$ORIGIN {{.Name}}.
; SOA record
; owner-name ttl class rr      name-server      email-addr  (sn ref ret ex min)
@                 IN   SOA     ns1.{{.Name}}.   root.{{.Name}}. (
			2017031088 ; sn = serial number
			3600       ; ref = refresh = 20m
			180        ; uret = update retry = 1m
			1209600    ; ex = expiry = 2w
			10800      ; nx = nxdomain ttl = 3h
			)
			NS	ns1.{{.Name}}.
; type syntax
; host ttl class type data
$ORIGIN {{.Name}}.
ns1                     A       192.168.199.129
{{range $k,$rr := .RRs}}{{$rr.Name}} {{$rr.TTL}} {{$rr.Type}} {{$rr.Value}}
{{end}}
