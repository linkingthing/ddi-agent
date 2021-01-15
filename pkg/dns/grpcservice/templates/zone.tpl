; zone file fragment for {{.Name}}

$TTL {{.TTL}}

$ORIGIN {{.Name}}
; SOA record
; owner-name ttl class rr      name-server      email-addr  (sn ref ret ex min)
@                 IN   SOA     {{.NSName}}   {{.RootName}} (
			2017031088 ; sn = serial number
			3600       ; ref = refresh = 20m
			180        ; uret = update retry = 1m
			1209600    ; ex = expiry = 2w
			10800      ; nx = nxdomain ttl = 3h
			)
			NS	{{.NSName}}
; type syntax
; host ttl class type data
$ORIGIN {{.Name}}
ns                     A       127.0.0.1
{{range $k,$rr := .RRs}}{{$rr.Name}} {{$rr.TTL}} {{$rr.Type}} {{$rr.Rdata}}
{{end}}
