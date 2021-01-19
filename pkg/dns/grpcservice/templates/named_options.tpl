options {
	directory "{{.ConfigPath}}";
	pid-file "named.pid";
	allow-new-zones yes;
	check-names master ignore;
	listen-on {any;};
    listen-on-v6 {any;};{{if .TransferPort}}
    listen-on port {{.TransferPort}} {any;};
    listen-on-v6 port {{.TransferPort}} {any;};{{end}}
	allow-query {any;};
	{{if .DnssecEnable}}dnssec-enable yes;{{else}}dnssec-enable no;{{end}}
	dnssec-validation no;
	recursive-clients {{.RecursiveClients}};
	{{if .LogEnable}}querylog yes;{{else}}querylog no;{{end}}{{if .BlackholeEnable}}
	blackhole{ {{range $k,$v := .Blackholes}}{{$v}}; {{end}}};{{end}}
	{{if .RecursionEnable}}recursion yes;{{else}}recursion no;{{end}}
};

statistics-channels {
     inet 0.0.0.0 port 58082;
};

{{if .LogEnable}}logging{
	channel query_log{
	buffered true;
	file "query.log" versions 5 size 200m;
	print-time yes;
	print-category yes;
        severity dynamic;
	};
	category queries{
	query_log;
	};
};{{end}}
