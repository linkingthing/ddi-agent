$TTL 300
@ IN SOA ns.example.net hostmaster.example.net 0 0 0 0 0
@ IN NS ns.example.net
;
; NS records do not need address records in this zone as it is not in the
; normal namespace.
;
{{range $k,$rr := .RRs}}{{$rr.Name}} {{$rr.TTL}} {{$rr.Type}} {{$rr.Value}}
{{end}}
