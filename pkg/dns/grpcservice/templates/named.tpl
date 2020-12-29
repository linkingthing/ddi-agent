key "rndc-key" {
	algorithm hmac-sha256;
	secret "c3Ryb25nIGVub3VnaCBmb3IgYSBtYW4gYnV0IG1hZGUgZm9yIGEgd29tYW4K";
};

controls {
        inet 127.0.0.1 port 953
        allow { 127.0.0.1; } keys { "rndc-key"; };
};

include "{{$.NamedAclPath}}";
include "{{$.NamedViewPath}}";
include "{{$.NamedOptionsPath}}";
