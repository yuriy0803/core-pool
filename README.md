# core-pool

## Features

* Support for HTTP and Stratum mining
* Detailed block stats with luck percentage and full reward
* Failover geth instances: geth high availability built in
* Separate stats for workers: can highlight timed-out workers so miners can perform maintenance of rigs
* JSON-API for stats
* New vue based UI
* Supports Ethereum Classic, Mordor, Ethereum, Ropsten & Ubiq.

## Building on Linux

Dependencies:

  * go >= 1.13
  * core-geth
  * redis-server >= 2.8.0
  * nodejs >= 4 LTS
  * nginx

**I highly recommend to use Ubuntu 20.04 LTS.**

First install  [core-geth](https://github.com/frkhash/core-geth/releases).

Clone & compile:

    git config --global http.https://gopkg.in.followRedirects true
    git clone https://github.com/frkhash/core-pool.git
    cd core-pool
    make

Install redis-server:

    sudo apt update
    sudo apt install redis-server

*The supervised directive is set to no by default. Since you are running Ubuntu, which uses the systemd init system, change this to systemd:*

    sudo nano /etc/redis/redis.conf

```bash
. . .

# If you run Redis from upstart or systemd, Redis can interact with your
# supervision tree. Options:
#   supervised no      - no supervision interaction
#   supervised upstart - signal upstart by putting Redis into SIGSTOP mode
#   supervised systemd - signal systemd by writing READY=1 to $NOTIFY_SOCKET
#   supervised auto    - detect upstart or systemd method based on
#                        UPSTART_JOB or NOTIFY_SOCKET environment variables
# Note: these supervision methods only signal "process is ready."
#       They do not enable continuous liveness pings back to your supervisor.
supervised systemd

. . .
```

    sudo systemctl restart redis.service


**make sure redis is bound to localhost**

  sudo nano /etc/redis/redis.conf

Locate this line and make sure it is uncommented (remove the # if it exists):

```bash
bind 127.0.0.1 ::1
```

Restart redis

    sudo systemctl restart redis

## Running Pool

    ./build/bin/core-pool config.json

You can use Ubuntu upstart - check for sample config in <code>upstart.conf</code>.

### Serving API using nginx

Create an upstream for API:

    upstream api {
        server 127.0.0.1:8080;
    }

and add this setting after <code>location /</code>:

    location /api {
        proxy_pass http://api;
    }

### Configuration

Configuration is actually simple, just read it twice and think twice before changing defaults.

**Don't copy config directly from this manual. Use the example config from the package,
otherwise you will get errors on start because of JSON comments.**

```javascript
{
  // Set to the number of CPU cores of your server
  "threads": 2,
  // Prefix for keys in redis store
  "coin": "etc",
  // Give unique name to each instance
  "name": "main",
  // mordor, classic, ethereum, ropsten or ubiq
  "network": "classic",
  "proxy": {
    "enabled": true,

    // Bind HTTP mining endpoint to this IP:PORT
    "listen": "0.0.0.0:8888",

    // Allow only this header and body size of HTTP request from miners
    "limitHeadersSize": 1024,
    "limitBodySize": 256,

    /* Set to true if you are behind CloudFlare (not recommended) or behind http-reverse
      proxy to enable IP detection from X-Forwarded-For header.
      Advanced users only. It's tricky to make it right and secure.
    */
    "behindReverseProxy": false,

    // Stratum mining endpoint
    "stratum": {
      "enabled": true,
      // Bind stratum mining socket to this IP:PORT
      "listen": "0.0.0.0:8008",
      "timeout": "120s",
      "maxConn": 8192,
      "tls": false,
      "certFile": "/path/to/cert.pem",
      "keyFile": "/path/to/key.pem"
    },

    // Try to get new job from node in this interval
    "blockRefreshInterval": "120ms",
    "stateUpdateInterval": "3s",
    // Require this share difficulty from miners
    "difficulty": 2000000000,

    /* Reply error to miner instead of job if redis is unavailable.
      Should save electricity to miners if pool is sick and they didn't set up failovers.
    */
    "healthCheck": true,
    // Mark pool sick after this number of redis failures.
    "maxFails": 100,
    // TTL for workers stats, usually should be equal to large hashrate window from API section
    "hashrateExpiration": "3h",

    "policy": {
      "workers": 8,
      "resetInterval": "60m",
      "refreshInterval": "1m",

      "banning": {
        "enabled": false,
        /* Name of ipset for banning.
        Check http://ipset.netfilter.org/ documentation.
        */
        "ipset": "blacklist",
        // Remove ban after this amount of time
        "timeout": 1800,
        // Percent of invalid shares from all shares to ban miner
        "invalidPercent": 30,
        // Check after after miner submitted this number of shares
        "checkThreshold": 30,
        // Bad miner after this number of malformed requests
        "malformedLimit": 5
      },
      // Connection rate limit
      "limits": {
        "enabled": false,
        // Number of initial connections
        "limit": 30,
        "grace": "5m",
        // Increase allowed number of connections on each valid share
        "limitJump": 10
      }
    }
  },

  // Provides JSON data for frontend which is static website
  "api": {
    "enabled": true,
    "listen": "0.0.0.0:8080",
    // Collect miners stats (hashrate, ...) in this interval
    "statsCollectInterval": "5s",
    // Purge stale stats interval
    "purgeInterval": "10m",
    // Fast hashrate estimation window for each miner from it's shares
    "hashrateWindow": "30m",
    // Long and precise hashrate from shares, 3h is cool, keep it
    "hashrateLargeWindow": "3h",
    // Collect stats for shares/diff ratio for this number of blocks
    "luckWindow": [64, 128, 256],
    // Max number of payments to display in frontend
    "payments": 50,
    // Max numbers of blocks to display in frontend
    "blocks": 50,

    /* If you are running API node on a different server where this module
      is reading data from redis writeable slave, you must run an api instance with this option enabled in order to purge hashrate stats from main redis node.
      Only redis writeable slave will work properly if you are distributing using redis slaves.
      Very advanced. Usually all modules should share same redis instance.
    */
    "purgeOnly": false
  },

  // Check health of each node in this interval
  "upstreamCheckInterval": "5s",

  /* List of parity nodes to poll for new jobs. Pool will try to get work from
    first alive one and check in background for failed to back up.
    Current block template of the pool is always cached in RAM indeed.
  */
  "upstream": [
    {
      "name": "main",
      "url": "http://127.0.0.1:8545",
      "timeout": "10s"
    },
    {
      "name": "backup",
      "url": "http://127.0.0.2:8545",
      "timeout": "10s"
    }
  ],

  // This is standard redis connection options
  "redis": {
    // Where your redis instance is listening for commands
    "endpoint": "127.0.0.1:6379",
    "poolSize": 10,
    "database": 0,
    "password": ""
  },

  // This module periodically remits ether to miners
  "unlocker": {
    "enabled": false,
    // Pool fee percentage
    "poolFee": 1.0,
    // Pool fees beneficiary address (leave it blank to disable fee withdrawals)
    "poolFeeAddress": "",
    // Donate 10% from pool fees to developers
    "donate": true,
    // Unlock only if this number of blocks mined back
    "depth": 120,
    // Simply don't touch this option
    "immatureDepth": 20,
    // Keep mined transaction fees as pool fees
    "keepTxFees": false,
    // Run unlocker in this interval
    "interval": "10m",
    // Parity node rpc endpoint for unlocking blocks
    "daemon": "http://127.0.0.1:8545",
    // Rise error if can't reach parity
    "timeout": "10s"
  },

  // Pay out miners using this module
  "payouts": {
    "enabled": false,
    // Require minimum number of peers on node
    "requirePeers": 25,
    // Run payouts in this interval
    "interval": "12h",
    // Parity node rpc endpoint for payouts processing
    "daemon": "http://127.0.0.1:8545",
    // Rise error if can't reach parity
    "timeout": "10s",
    // Address with pool balance
    "address": "0x0",
    // Let parity to determine gas and gasPrice
    "autoGas": true,
    // Gas amount and price for payout tx (advanced users only)
    "gas": "21000",
    "gasPrice": "50000000000",
    // Send payment only if miner's balance is >= 0.5 Ether
    "threshold": 500000000,
    // Perform BGSAVE on Redis after successful payouts session
    "bgsave": false
  }
}
```

If you are distributing your pool deployment to several servers or processes,
create several configs and disable unneeded modules on each server. (Advanced users)

I recommend this deployment strategy:

* Mining instance - 1x (it depends, you can run one node for EU, one for US, one for Asia)
* Unlocker and payouts instance - 1x each (strict!)
* API instance - 1x

### Notes

* Unlocking and payouts are sequential, 1st tx go, 2nd waiting for 1st to confirm and so on. You can disable that in code. Carefully read `docs/PAYOUTS.md`.
* Also, keep in mind that **unlocking and payouts will halt in case of backend or node RPC errors**. In that case check everything and restart.
* You must restart module if you see errors with the word *suspended*.
* Don't run payouts and unlocker modules as part of mining node. Create separate configs for both, launch independently and make sure you have a single instance of each module running.
* If `poolFeeAddress` is not specified all pool profit will remain on coinbase address. If it specified, make sure to periodically send some dust back required for payments.

### frontend

See https://github.com/frkhash/core-pool-interface
