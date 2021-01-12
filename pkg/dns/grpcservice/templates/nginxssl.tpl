server {
    listen              443 ssl;
    server_name         {{.Domain}};
    ssl_certificate     conf.d/key/{{.Domain}}.crt;
    ssl_certificate_key conf.d/key/{{.Domain}}.key;
    ssl_protocols       TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    location / {
            proxy_pass {{.Url}};
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header Host $http_host;
            proxy_set_header X-NginX-Proxy true;
            proxy_buffering    off;
            proxy_buffer_size  512k;
            proxy_buffers 10  512k;
            client_max_body_size 100m;
    }

    location = /50x.html {
        root   /usr/share/nginx/html;
    }
}
