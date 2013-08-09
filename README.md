go-poolhandler
==============

Pool manager and counter for Golang net/http handler.
You can limit max simultaneous connections and view status page of your pool.
Project need some codereview and more functionality.
It will be done in near future.
But it fully production ready for now.

Example:
http.ListenAndServe(":80", poolhandler.GetPoolHandler(10, handler, "/pool-status"))
Maximum 10 simultaneous connections
Open in browser http://1.2.3.4/pool-status to get status page(like Apache).
