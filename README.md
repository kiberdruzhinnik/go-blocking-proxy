# go-blocking-proxy

A minimal SOCKS5 endpoint for network blocking. It negotiates SOCKS5, then rejects every `CONNECT` request with `connection not allowed`. It never opens an upstream connection, resolves a requested hostname, or forwards traffic. This makes it useful as a SOCKS endpoint to which a router can redirect selected clients or domains.

## Run

```sh
go run . -listen 192.168.1.2:1080
```

Replace the address with this host's LAN address. `-listen :1080` binds on all interfaces. The default is `127.0.0.1:1080`, which is only reachable from the local machine.

Build a binary with:

```sh
go build -o go-blocking-proxy .
./go-blocking-proxy -listen 192.168.1.2:1080
```

Configure the router or client to use this endpoint as its SOCKS5 proxy for the traffic you want blocked. SOCKS5 clients normally receive an explicit connection failure; some applications may fall back to a direct connection, so enforce the policy at the router firewall as well if bypass must be prevented.

The server supports the no-authentication SOCKS5 method and `CONNECT`. It rejects other commands. Restrict the listening interface and firewall access to trusted LAN clients.
