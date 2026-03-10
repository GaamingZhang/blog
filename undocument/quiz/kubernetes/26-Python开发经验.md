---
date: 2026-03-10
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Python
tag:
  - Python
  - 运维开发
---

# 运维Python开发实践

## 为什么运维需要Python？

Python因其简洁的语法和丰富的库，成为运维自动化的首选语言。

```
┌─────────────────────────────────────────────────────────────┐
│                    Python运维应用场景                        │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              自动化脚本                              │    │
│  │  - 服务器巡检                                       │    │
│  │  - 批量操作                                         │    │
│  │  - 日志分析                                         │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              工具开发                                │    │
│  │  - CLI工具                                          │    │
│  │  - Web API                                          │    │
│  │  - 监控Exporter                                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              数据处理                                │    │
│  │  - 数据采集                                         │    │
│  │  - 数据清洗                                         │    │
│  │  - 报表生成                                         │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 运维自动化脚本

### 服务器巡检脚本

```python
#!/usr/bin/env python3
import subprocess
import json
from datetime import datetime

class ServerInspector:
    def __init__(self):
        self.results = {}
    
    def check_cpu(self):
        result = subprocess.run(
            ['top', '-bn1'],
            capture_output=True,
            text=True
        )
        lines = result.stdout.split('\n')
        cpu_line = lines[2]
        cpu_usage = 100 - float(cpu_line.split()[7].replace('%', ''))
        self.results['cpu_usage'] = round(cpu_usage, 2)
    
    def check_memory(self):
        with open('/proc/meminfo') as f:
            meminfo = {}
            for line in f:
                key, value = line.split(':')
                meminfo[key.strip()] = int(value.strip().split()[0])
        
        total = meminfo['MemTotal']
        available = meminfo['MemAvailable']
        used = total - available
        usage_percent = (used / total) * 100
        
        self.results['memory'] = {
            'total_gb': round(total / 1024 / 1024, 2),
            'used_gb': round(used / 1024 / 1024, 2),
            'available_gb': round(available / 1024 / 1024, 2),
            'usage_percent': round(usage_percent, 2)
        }
    
    def check_disk(self):
        result = subprocess.run(
            ['df', '-h'],
            capture_output=True,
            text=True
        )
        lines = result.stdout.split('\n')[1:]
        disks = []
        for line in lines:
            if line:
                parts = line.split()
                if len(parts) >= 6:
                    disks.append({
                        'filesystem': parts[0],
                        'size': parts[1],
                        'used': parts[2],
                        'available': parts[3],
                        'usage': parts[4],
                        'mount': parts[5]
                    })
        self.results['disks'] = disks
    
    def run(self):
        self.check_cpu()
        self.check_memory()
        self.check_disk()
        self.results['timestamp'] = datetime.now().isoformat()
        return self.results

if __name__ == '__main__':
    inspector = ServerInspector()
    results = inspector.run()
    print(json.dumps(results, indent=2))
```

### Kubernetes资源清理脚本

```python
#!/usr/bin/env python3
from kubernetes import client, config
from datetime import datetime, timedelta

class KubernetesCleaner:
    def __init__(self):
        config.load_kube_config()
        self.v1 = client.CoreV1Api()
        self.batch_v1 = client.BatchV1Api()
    
    def clean_completed_pods(self, namespace='default'):
        pods = self.v1.list_namespaced_pod(namespace)
        cleaned = 0
        
        for pod in pods.items:
            if pod.status.phase == 'Succeeded':
                self.v1.delete_namespaced_pod(pod.metadata.name, namespace)
                cleaned += 1
                print(f"Deleted completed pod: {pod.metadata.name}")
        
        return cleaned
    
    def clean_evicted_pods(self, namespace='default'):
        pods = self.v1.list_namespaced_pod(namespace)
        cleaned = 0
        
        for pod in pods.items:
            if pod.status.reason == 'Evicted':
                self.v1.delete_namespaced_pod(pod.metadata.name, namespace)
                cleaned += 1
                print(f"Deleted evicted pod: {pod.metadata.name}")
        
        return cleaned
    
    def clean_old_jobs(self, namespace='default', days=7):
        jobs = self.batch_v1.list_namespaced_job(namespace)
        cleaned = 0
        cutoff = datetime.now() - timedelta(days=days)
        
        for job in jobs.items:
            if job.status.completion_time:
                if job.status.completion_time.replace(tzinfo=None) < cutoff:
                    self.batch_v1.delete_namespaced_job(
                        job.metadata.name,
                        namespace
                    )
                    cleaned += 1
                    print(f"Deleted old job: {job.metadata.name}")
        
        return cleaned

if __name__ == '__main__':
    cleaner = KubernetesCleaner()
    cleaner.clean_completed_pods()
    cleaner.clean_evicted_pods()
    cleaner.clean_old_jobs()
```

## Web API开发

### FastAPI示例

```python
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional
import uvicorn

app = FastAPI(title="Server Management API")

class Server(BaseModel):
    hostname: str
    ip: str
    environment: str
    status: Optional[str] = "active"

servers_db = {}

@app.get("/servers")
async def list_servers():
    return {"servers": list(servers_db.values())}

@app.get("/servers/{hostname}")
async def get_server(hostname: str):
    if hostname not in servers_db:
        raise HTTPException(status_code=404, detail="Server not found")
    return servers_db[hostname]

@app.post("/servers")
async def create_server(server: Server):
    if server.hostname in servers_db:
        raise HTTPException(status_code=400, detail="Server already exists")
    servers_db[server.hostname] = server.dict()
    return {"message": "Server created", "server": server}

@app.put("/servers/{hostname}")
async def update_server(hostname: str, server: Server):
    if hostname not in servers_db:
        raise HTTPException(status_code=404, detail="Server not found")
    servers_db[hostname] = server.dict()
    return {"message": "Server updated", "server": server}

@app.delete("/servers/{hostname}")
async def delete_server(hostname: str):
    if hostname not in servers_db:
        raise HTTPException(status_code=404, detail="Server not found")
    del servers_db[hostname]
    return {"message": "Server deleted"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

## CLI工具开发

### 使用Click开发命令行工具

```python
import click
import requests

@click.group()
def cli():
    """Server Management CLI Tool"""
    pass

@cli.command()
@click.option('--name', required=True, help='Server name')
@click.option('--ip', required=True, help='Server IP address')
@click.option('--env', default='dev', help='Environment')
def create(name, ip, env):
    """Create a new server"""
    data = {
        'hostname': name,
        'ip': ip,
        'environment': env
    }
    response = requests.post('http://localhost:8000/servers', json=data)
    click.echo(f"Server created: {response.json()}")

@cli.command()
@click.option('--name', required=True, help='Server name')
def get(name):
    """Get server details"""
    response = requests.get(f'http://localhost:8000/servers/{name}')
    if response.status_code == 200:
        click.echo(response.json())
    else:
        click.echo("Server not found", err=True)

@cli.command()
def list():
    """List all servers"""
    response = requests.get('http://localhost:8000/servers')
    for server in response.json()['servers']:
        click.echo(f"- {server['hostname']} ({server['ip']})")

@cli.command()
@click.option('--name', required=True, help='Server name')
def delete(name):
    """Delete a server"""
    response = requests.delete(f'http://localhost:8000/servers/{name}')
    if response.status_code == 200:
        click.echo(f"Server {name} deleted")
    else:
        click.echo("Server not found", err=True)

if __name__ == '__main__':
    cli()
```

## 日志分析脚本

```python
import re
from collections import Counter
from datetime import datetime

class LogAnalyzer:
    def __init__(self, log_file):
        self.log_file = log_file
        self.pattern = re.compile(
            r'(?P<ip>\S+) \S+ \S+ \[(?P<time>[^\]]+)\] '
            r'"(?P<method>\S+) (?P<path>\S+) \S+" '
            r'(?P<status>\d+) (?P<size>\S+)'
        )
    
    def parse_line(self, line):
        match = self.pattern.match(line)
        if match:
            return match.groupdict()
        return None
    
    def analyze(self):
        status_codes = Counter()
        paths = Counter()
        ips = Counter()
        
        with open(self.log_file, 'r') as f:
            for line in f:
                parsed = self.parse_line(line)
                if parsed:
                    status_codes[parsed['status']] += 1
                    paths[parsed['path']] += 1
                    ips[parsed['ip']] += 1
        
        return {
            'status_codes': dict(status_codes.most_common(10)),
            'top_paths': dict(paths.most_common(10)),
            'top_ips': dict(ips.most_common(10))
        }
    
    def generate_report(self):
        analysis = self.analyze()
        report = []
        report.append("=== Log Analysis Report ===\n")
        
        report.append("\nTop Status Codes:")
        for code, count in analysis['status_codes'].items():
            report.append(f"  {code}: {count}")
        
        report.append("\nTop Paths:")
        for path, count in analysis['top_paths'].items():
            report.append(f"  {path}: {count}")
        
        report.append("\nTop IPs:")
        for ip, count in analysis['top_ips'].items():
            report.append(f"  {ip}: {count}")
        
        return '\n'.join(report)

if __name__ == '__main__':
    analyzer = LogAnalyzer('/var/log/nginx/access.log')
    print(analyzer.generate_report())
```

## Prometheus Exporter开发

```python
from prometheus_client import start_http_server, Gauge, Counter
import random
import time

class CustomExporter:
    def __init__(self, port=8000):
        self.port = port
        
        self.cpu_usage = Gauge(
            'app_cpu_usage_percent',
            'CPU usage percentage'
        )
        
        self.memory_usage = Gauge(
            'app_memory_usage_bytes',
            'Memory usage in bytes'
        )
        
        self.request_count = Counter(
            'app_requests_total',
            'Total requests',
            ['method', 'endpoint']
        )
        
        self.request_latency = Gauge(
            'app_request_latency_seconds',
            'Request latency in seconds',
            ['endpoint']
        )
    
    def collect_metrics(self):
        self.cpu_usage.set(random.uniform(0, 100))
        self.memory_usage.set(random.uniform(0, 8 * 1024 * 1024 * 1024))
        self.request_count.labels(method='GET', endpoint='/api/users').inc()
        self.request_latency.labels(endpoint='/api/users').set(random.uniform(0.01, 1))
    
    def run(self):
        start_http_server(self.port)
        print(f"Exporter running on port {self.port}")
        
        while True:
            self.collect_metrics()
            time.sleep(15)

if __name__ == '__main__':
    exporter = CustomExporter(port=8000)
    exporter.run()
```

## 最佳实践

### 1. 代码规范

```python
from typing import List, Dict, Optional

def process_data(data: List[Dict]) -> Dict:
    """
    Process a list of data items.
    
    Args:
        data: List of dictionaries to process
    
    Returns:
        Dictionary with processing results
    
    Raises:
        ValueError: If data is empty
    """
    if not data:
        raise ValueError("Data cannot be empty")
    return {"count": len(data)}
```

### 2. 异常处理

```python
import logging

logger = logging.getLogger(__name__)

def safe_operation():
    try:
        result = risky_operation()
    except ConnectionError as e:
        logger.error(f"Connection failed: {e}")
        raise
    except ValueError as e:
        logger.warning(f"Invalid value: {e}")
        return None
    else:
        logger.info("Operation successful")
        return result
    finally:
        cleanup()
```

### 3. 配置管理

```python
from pydantic import BaseSettings

class Settings(BaseSettings):
    app_name: str = "MyApp"
    debug: bool = False
    database_url: str
    redis_url: str
    
    class Config:
        env_file = ".env"

settings = Settings()
```

### 4. 日志记录

```python
import logging
from logging.handlers import RotatingFileHandler

def setup_logging():
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.INFO)
    
    handler = RotatingFileHandler(
        'app.log',
        maxBytes=10*1024*1024,
        backupCount=5
    )
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    handler.setFormatter(formatter)
    logger.addHandler(handler)
    
    return logger
```

## 参考资源

- [Python官方文档](https://docs.python.org/3/)
- [FastAPI文档](https://fastapi.tiangolo.com/)
- [Click文档](https://click.palletsprojects.com/)
