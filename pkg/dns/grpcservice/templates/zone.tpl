; zone file fragment for {{.Name}}

$TTL {{.TTL}}

$ORIGIN {{.Name}}
{{range $k,$rr := .RRs}}{{$rr.Name}} {{$rr.TTL}} {{$rr.Type}} {{$rr.Rdata}}
{{end}}