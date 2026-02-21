#!/bin/bash
# Test kube-doctor MCP server with JSON-RPC calls
cd "$(dirname "$0")"

printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}\n{"jsonrpc":"2.0","method":"notifications/initialized"}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}\n{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_namespaces","arguments":{}}}\n{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"cluster_info","arguments":{}}}\n{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_nodes","arguments":{}}}\n' | ./kube-doctor 2>/dev/null
