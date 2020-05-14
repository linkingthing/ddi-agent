$TTL 7200	; 2 hours
@			IN SOA	nons.blocked.com. noemail.blocked.com. (
				2018031408 ; serial
				43200      ; refresh (12 hours)
				900        ; retry (15 minutes)
				1814400    ; expire (3 weeks)
				7200       ; minimum (2 hours)
				)
@       IN              NS      nons.blocked.com.

{{range $k,$rr := .RRs}}{{$rr.Name}} {{$rr.TTL}} {{$rr.Type}} {{$rr.Value}}
{{end}}
