# New zone file for view: {{.ViewName}}
# This file contains configuration for zones added by
# the 'rndc addzone' command. DO NOT EDIT BY HAND.
{{$view := .ViewName}}{{range $k, $zone := .Zones}}zone "{{$zone.Name}}" in {{$view}} { type master; file "{{$zone.ZoneFile}}"; };{{end}}
