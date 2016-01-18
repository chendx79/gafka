global    
    # logging to rsyslog facility local3 [err warning info debug]   
    log 127.0.0.1 local3 info

    maxconn  51200
    ulimit-n 102400
    pidfile {{.HaproxyRoot}}/haproxy.pid
    daemon
    nbproc {{.CpuNum}}
    spread-checks 5
    #user  haproxy
    #group haproxy
    #chroot {{.HaproxyRoot}}

defaults
    log global
    mode http # [tcp|http|health]
    backlog 10000
    retries 0
    maxconn 15000
    balance roundrobin
    
    no option httpclose
    option httplog
    option dontlognull  # 不记录健康检查的日志信息
    option abortonclose # 当服务器负载很高的时候，自动结束掉当前队列处理比较久的链接
    option redispatch   # 当服务器组中的某台设备故障后，自动将请求重定向到组内其他主机
    option forwardfor   # X-Forwarded-For: remote client ip
    
    timeout client          10m  # 客户端侧最大非活动时间
    timeout server          1m   # 服务器侧最大非活动时间
    timeout connect         10s  # 连接服务器超时时间
    timeout http-keep-alive 6m   # ?
    timeout queue           1m   # 一个请求在队列里的超时时间
    timeout check           5s
    #timeout http-request    5s

    default-server weight 10 minconn 50 maxconn 5000 inter 30s rise 2 fall 3
    
    option log-separate-errors
    errorfile 400 {{.LogDir}}/400.http
    errorfile 500 {{.LogDir}}/500.http
    errorfile 502 {{.LogDir}}/502.http
    errorfile 503 {{.LogDir}}/503.http
    errorfile 504 {{.LogDir}}/504.http

listen dashboard
    bind 0.0.0.0:10890
    mode http
    stats refresh 30s
    stats uri /stats
    stats realm Haproxy Manager
    stats auth admin:admin

listen pub
    bind 0.0.0.0:10891
    cookie PUB insert indirect
    option httpchk GET /alive HTTP/1.1\r\nHost:pub.ffan.com
{{range .Pub}}
    server {{.Name}} {{.Addr}} cookie {{.Name}} check
{{end}}

listen sub
    bind 0.0.0.0:10892
    balance uri
    #mode tcp
    cookie SUB insert indirect
    option httpchk GET /alive HTTP/1.1\r\nHost:sub.ffan.com
{{range .Sub}}
    server {{.Name}} {{.Addr}} cookie {{.Name}} check
{{end}}

listen man
    bind 0.0.0.0:10893
    option httpchk GET /alive HTTP/1.1\r\nHost:kman.ffan.com
{{range .Man}}
    server {{.Name}} {{.Addr}} check
{{end}}