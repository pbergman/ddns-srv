# DDNS Server

ddns-srv is a dynamic DNS server written in Go that bridges the gap between the [DynDNS](https://help.dyn.com/perform-update.html) 
protocol and hosting providers that do not natively support updating DNS records via DDNS.

It leverages [libdns](https://github.com/libdns/libdns) to manage DNS records across providers. Providers can be easily 
integrated as plugins by simply exporting a few variables.

While Go plugins have certain drawbacks, in this case the flexibility and simplicity they provide outweigh the limitations 
and making it easy to create, replace, or upgrade providers without rebuilding or reinstalling the main application.

### Query Records

The server also has some url`s for checking records managed by the registered providers. It has three endpoints on which 
you can request to get some information. 

   - **/zones** this will print all available zones provided by all registered plugins.
   - **/lookup/\<type\>/\<hostname\>** type can be omitted and will default to A, this will return the data field of matching records
   - any other request will print all available records grouped by provider 

```bash
~: curl 'http://foo:bar@127.0.0.1:8080/zones'

• github.com/libdns/example
└─ example.com.

```

```bash
~: curl 'http://foo:bar@127.0.0.1:8080/lookup/local.example.com'

127.0.0.1

```

```bash
~: curl 'http://foo:bar@127.0.0.1:8080'

┌─────────────────────────────────────────────────────────────────────────────┐
│ github.com/libdns/example                                                   │
├───────────────────────────────────┬───────┬────────┬────────────────────────┤
│ Name                              │ Type  │ TTL    │ Data                   │
├───────────────────────────────────┼───────┼────────┼────────────────────────┤
│ local.example.com.                │ A     │ 1h0m0s │ 127.0.0.1              │
└───────────────────────────────────┴───────┴────────┴────────────────────────┘
```



### Configuration

At the moment we only support `json` config and perhaps this will change but for now it was easiest to configure the providers.


```

# The directory where the application looks for provider plugins.
# 
# Default: /usr/share/ddns-server
plugin_dir: 

server: {
   # The address the HTTP server will bind to for incoming requests.
   #
   # Defaults: :8080   
   listen:
   
   # To enable basic authentication, you can define a hash map where
   # the username is the key and the password is the value.When empty, 
   # the application will not validate any incoming requests. 
   users: {
      <name>: <password>
   }
   
   # When a request doesn’t include an IP or the format is invalid,
   # the application will use the client’s IP address.
   #
   # If ddns-srv is running behind a proxy (like Nginx or Caddy),
   # it can use the `X-Forwarded-For` headers. To ensure security,
   # the application validates the remote IP against a list of
   # trusted networks.
   #
   #   trusted_remotes: [
   #      "127.0.0.1/32"   
   #   ]
   #   
   # This means any requests originating from localhost are trusted,
   # and the first public IP in the `X-Forwarded-For` chain (from right to left)
   # will be used as the client’s IP.
   trusted_remotes: [
      <ip range>   
   ]
}

# Eeach plugin entry should include at least the module name. For providers 
# that do not support libdns.ZoneLister, you must also define the zones they 
# manage. All other fields are passed directly to the provider during 
# initialization.
# 
# So for a provider that does not suport the ZoneLister, but is responsible 
# for managing the example.com zone:
# 
# plugins: [
#   {
#      "module": "github.com/libdns/example"
#      "zones": [
#         "example.com"
#      ],
#      "api_key": "0ab1c2d3"
#   }
# ]
#
plugins: [
   {
      "module": <name as defined by the PluginModule varable in the pluigin>
   }
   ...
]

```

### Creating a Plugin

The easiest way is to use [ddns-srv-plugin-builder](https://github.com/pbergman/ddns-srv-plugin-builder) by installing

```bash
go install github.com/pbergman/ddns-srv-plugin-builder@latest
```

And just the necessary plugins like:

```bash
ddns-srv-plugin-builder github.com/libdns/provider1 github.com/libdns/provider2.... 
```

Or you can create them manually by following the steps below.

#### Manual Steps

1. **Create and initialize a new module:**

   ```bash
   mkdir example-plugin
   cd example-plugin
   go mod init plugins/example
   ```

2. **Add the desired libdns provider (for example `example`):**

   ```bash
   go get github.com/libdns/example
   ```

3. **Create a `main.go` file** with the following contents:

   ```go
   package main

   import (
       "github.com/libdns/example"
   )

   var (
       Plugin        *example.Provider
       PluginModule  = "github.com/libdns/example"
       PluginVersion = "v1.0.0"
   )
   ```

4. **Build the plugin:**

   ```bash
   go build -buildmode=plugin -o example.so
   ```

This will generate a `example.so` plugin file that can be used by `ddns-srv`.


### Example Configuration (Vyatta / EdgeOS)

For EdgeRouters or any operating system based on **Vyatta OS**, the following commands configure the DDNS client to update the DNS records for the domains `bar.example.com` and `foo.example.com` using the ip on network interface **`eth1`**.

The DDNS client communicates with your running `ddns-srv` instance at `https://10.0.0.101`.

```bash
configure
set service dns dynamic interface eth1 service custom-ddns_bridge host-name bar.example.com
set service dns dynamic interface eth1 service custom-ddns_bridge host-name foo.example.com
set service dns dynamic interface eth1 service custom-ddns_bridge protocol dyndns2
set service dns dynamic interface eth1 service custom-ddns_bridge login foo
set service dns dynamic interface eth1 service custom-ddns_bridge password bar
set service dns dynamic interface eth1 service custom-ddns_bridge server 10.0.0.101
commit; save; exit
```

#### Disable SSL (https)

By default, the automatically generated configuration will set `ssl=yes`. If you are running the service within a local network, you may want to disable SSL.

You can do this by updating the config manually:

```
sed -i -e 's/ssl=yes/ssl=no/g' /etc/ddclient/ddclient_eth1.conf
```

Now it will do an update on `http://10.0.0.101`

#### Force an update

To manually trigger an update of the DDNS records for `eth1`, run:

```bash
update dns dynamic interface eth1
```

#### Show DDNS Status

To check the current DDNS status and update timestamps, run:

```bash
show dns dynamic status
```
