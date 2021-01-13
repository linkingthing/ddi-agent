# New zone file for view: {{.ViewName}}
# This file contains configuration for zones added by
# the 'rndc addzone' command. DO NOT EDIT BY HAND.
{{$view := .ViewName}}{{range $k, $zone := .Zones}}
zone "{{$zone.Name}}" in {{$view}} { type {{$zone.Role}}; file "{{$zone.ZoneFile}}"; also-notify { {{$zone.Slaves}} }; masters { {{$zone.Masters}} };};{{end}}