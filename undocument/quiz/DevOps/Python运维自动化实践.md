---
date: 2026-03-11
author: Jiaming Zhang
isOriginal: false
article: true
category:
  - Automation
tag:
  - Python
  - Automation
  - DevOps
---

# Python 运维自动化实践

当你需要每天检查 100 台服务器的磁盘使用率、批量更新 50 个应用的配置、定时备份 20 个数据库时,手动操作不仅耗时而且容易出错。Python 作为运维自动化的首选语言,其丰富的库生态和简洁的语法使其成为提升运维效率的利器。但 Python 自动化并不是"写个脚本就行"——代码规范、错误处理、日志记录、性能优化都需要深入理解才能写出生产级别的自动化工具。

本文将从常用库与工具、脚本开发模式、自动化场景、性能优化、最佳实践五个维度,系统梳理 Python 运维自动化的实践经验。

## 一、常用库与工具

### 核心库概览

```
┌─────────────────────────────────────────────────────────────┐
│                    Python 运维自动化库                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  系统管理                                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - os: 操作系统接口                                   │  │
│  │  - sys: 系统参数                                      │  │
│  │  - subprocess: 进程管理                               │  │
│  │  - shutil: 文件操作                                   │  │
│  │  - pathlib: 路径处理                                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  网络操作                                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - requests: HTTP 客户端                              │  │
│  │  - paramiko: SSH 客户端                               │  │
│  │  - fabric: 远程执行                                   │  │
│  │  - netmiko: 网络设备自动化                            │  │
│  │  - socket: 底层网络                                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  云平台 SDK                                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - boto3: AWS SDK                                     │  │
│  │  - google-cloud: GCP SDK                              │  │
│  │  - azure-sdk: Azure SDK                               │  │
│  │  - kubernetes: K8s 客户端                             │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  数据处理                                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - pandas: 数据分析                                   │  │
│  │  - openpyxl: Excel 操作                               │  │
│  │  - PyYAML: YAML 解析                                  │  │
│  │  - Jinja2: 模板引擎                                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  任务调度                                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  - schedule: 定时任务                                 │  │
│  │  - celery: 分布式任务队列                             │  │
│  │  - APScheduler: 高级调度                              │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 系统管理

**subprocess 模块**:

```python
import subprocess
from typing import List, Tuple, Optional

def run_command(
    command: List[str],
    cwd: Optional[str] = None,
    timeout: int = 300,
    check: bool = True
) -> Tuple[int, str, str]:
    """
    执行 shell 命令
    
    Args:
        command: 命令列表
        cwd: 工作目录
        timeout: 超时时间(秒)
        check: 是否检查返回码
    
    Returns:
        (returncode, stdout, stderr)
    """
    try:
        result = subprocess.run(
            command,
            cwd=cwd,
            timeout=timeout,
            capture_output=True,
            text=True,
            check=check
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        raise TimeoutError(f"Command {command} timed out after {timeout} seconds")
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Command {command} failed with return code {e.returncode}: {e.stderr}")

# 使用示例
returncode, stdout, stderr = run_command(['kubectl', 'get', 'pods', '-n', 'production'])
print(stdout)
```

**pathlib 模块**:

```python
from pathlib import Path
import shutil

def backup_directory(source: str, destination: str) -> None:
    """
    备份目录
    
    Args:
        source: 源目录
        destination: 目标目录
    """
    source_path = Path(source)
    dest_path = Path(destination)
    
    if not source_path.exists():
        raise FileNotFoundError(f"Source directory {source} does not exist")
    
    # 创建目标目录
    dest_path.mkdir(parents=True, exist_ok=True)
    
    # 复制文件
    for item in source_path.glob('*'):
        if item.is_file():
            shutil.copy2(item, dest_path / item.name)
        elif item.is_dir():
            shutil.copytree(item, dest_path / item.name)

# 使用示例
backup_directory('/etc/nginx', '/backup/nginx')
```

### 网络操作

**requests 模块**:

```python
import requests
from typing import Dict, Any, Optional
import time

class HTTPClient:
    """HTTP 客户端封装"""
    
    def __init__(self, base_url: str, timeout: int = 30, retries: int = 3):
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        self.retries = retries
        self.session = requests.Session()
    
    def request(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict] = None,
        data: Optional[Dict] = None,
        headers: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """
        发送 HTTP 请求
        
        Args:
            method: HTTP 方法
            endpoint: 端点路径
            params: 查询参数
            data: 请求体
            headers: 请求头
        
        Returns:
            响应 JSON
        """
        url = f"{self.base_url}/{endpoint.lstrip('/')}"
        
        for attempt in range(self.retries):
            try:
                response = self.session.request(
                    method=method,
                    url=url,
                    params=params,
                    json=data,
                    headers=headers,
                    timeout=self.timeout
                )
                response.raise_for_status()
                return response.json()
            except requests.exceptions.RequestException as e:
                if attempt == self.retries - 1:
                    raise
                time.sleep(2 ** attempt)  # 指数退避
        
        return {}

# 使用示例
client = HTTPClient('https://api.example.com')
result = client.get('/users', params={'page': 1})
```

**paramiko 模块**:

```python
import paramiko
from typing import List, Optional

class SSHClient:
    """SSH 客户端封装"""
    
    def __init__(
        self,
        host: str,
        username: str,
        password: Optional[str] = None,
        key_file: Optional[str] = None,
        port: int = 22
    ):
        self.host = host
        self.username = username
        self.password = password
        self.key_file = key_file
        self.port = port
        self.client = None
    
    def connect(self) -> None:
        """建立 SSH 连接"""
        self.client = paramiko.SSHClient()
        self.client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        
        self.client.connect(
            hostname=self.host,
            port=self.port,
            username=self.username,
            password=self.password,
            key_filename=self.key_file
        )
    
    def execute(self, command: str) -> Tuple[int, str, str]:
        """
        执行远程命令
        
        Args:
            command: 命令字符串
        
        Returns:
            (returncode, stdout, stderr)
        """
        if not self.client:
            self.connect()
        
        stdin, stdout, stderr = self.client.exec_command(command)
        return (
            stdout.channel.recv_exit_status(),
            stdout.read().decode('utf-8'),
            stderr.read().decode('utf-8')
        )
    
    def upload_file(self, local_path: str, remote_path: str) -> None:
        """
        上传文件
        
        Args:
            local_path: 本地文件路径
            remote_path: 远程文件路径
        """
        if not self.client:
            self.connect()
        
        sftp = self.client.open_sftp()
        sftp.put(local_path, remote_path)
        sftp.close()
    
    def close(self) -> None:
        """关闭连接"""
        if self.client:
            self.client.close()

# 使用示例
ssh = SSHClient('server.example.com', 'user', key_file='~/.ssh/id_rsa')
returncode, stdout, stderr = ssh.execute('kubectl get pods -n production')
ssh.close()
```

### 云平台 SDK

**boto3 (AWS)**:

```python
import boto3
from typing import List, Dict, Any

class EC2Manager:
    """EC2 管理类"""
    
    def __init__(self, region: str = 'us-east-1'):
        self.ec2 = boto3.client('ec2', region_name=region)
    
    def list_instances(self, filters: Optional[List[Dict]] = None) -> List[Dict]:
        """
        列出 EC2 实例
        
        Args:
            filters: 过滤条件
        
        Returns:
            实例列表
        """
        params = {}
        if filters:
            params['Filters'] = filters
        
        instances = []
        response = self.ec2.describe_instances(**params)
        
        for reservation in response['Reservations']:
            instances.extend(reservation['Instances'])
        
        return instances
    
    def start_instance(self, instance_id: str) -> Dict:
        """
        启动实例
        
        Args:
            instance_id: 实例 ID
        
        Returns:
            响应结果
        """
        return self.ec2.start_instances(InstanceIds=[instance_id])
    
    def stop_instance(self, instance_id: str) -> Dict:
        """
        停止实例
        
        Args:
            instance_id: 实例 ID
        
        Returns:
            响应结果
        """
        return self.ec2.stop_instances(InstanceIds=[instance_id])

# 使用示例
ec2 = EC2Manager()
instances = ec2.list_instances(filters=[{'Name': 'tag:Environment', 'Values': ['production']}])
```

**kubernetes 客户端**:

```python
from kubernetes import client, config
from typing import List, Dict

class K8sManager:
    """Kubernetes 管理类"""
    
    def __init__(self, kubeconfig: Optional[str] = None):
        if kubeconfig:
            config.load_kube_config(config_file=kubeconfig)
        else:
            config.load_kube_config()
        
        self.v1 = client.CoreV1Api()
        self.apps_v1 = client.AppsV1Api()
    
    def list_pods(self, namespace: str = 'default') -> List[Dict]:
        """
        列出 Pod
        
        Args:
            namespace: 命名空间
        
        Returns:
            Pod 列表
        """
        pods = self.v1.list_namespaced_pod(namespace)
        return [
            {
                'name': pod.metadata.name,
                'namespace': pod.metadata.namespace,
                'status': pod.status.phase,
                'ip': pod.status.pod_ip,
                'node': pod.spec.node_name
            }
            for pod in pods.items
        ]
    
    def scale_deployment(
        self,
        name: str,
        namespace: str,
        replicas: int
    ) -> Dict:
        """
        扩缩容 Deployment
        
        Args:
            name: Deployment 名称
            namespace: 命名空间
            replicas: 副本数
        
        Returns:
            响应结果
        """
        body = {'spec': {'replicas': replicas}}
        return self.apps_v1.patch_namespaced_deployment_scale(
            name=name,
            namespace=namespace,
            body=body
        )

# 使用示例
k8s = K8sManager()
pods = k8s.list_pods(namespace='production')
k8s.scale_deployment('my-app', 'production', 5)
```

## 二、脚本开发模式

### 配置管理

**YAML 配置**:

```yaml
# config.yaml
database:
  host: localhost
  port: 5432
  username: admin
  password: password
  database: production

servers:
  - host: server1.example.com
    username: user
    key_file: ~/.ssh/id_rsa
  - host: server2.example.com
    username: user
    key_file: ~/.ssh/id_rsa

tasks:
  - name: check_disk_usage
    schedule: "0 9 * * *"
    enabled: true
  - name: backup_database
    schedule: "0 2 * * *"
    enabled: true
```

```python
import yaml
from pathlib import Path
from typing import Dict, Any

class ConfigManager:
    """配置管理类"""
    
    def __init__(self, config_file: str):
        self.config_file = Path(config_file)
        self.config = self._load_config()
    
    def _load_config(self) -> Dict[str, Any]:
        """加载配置"""
        if not self.config_file.exists():
            raise FileNotFoundError(f"Config file {self.config_file} not found")
        
        with open(self.config_file, 'r') as f:
            return yaml.safe_load(f)
    
    def get(self, key: str, default: Any = None) -> Any:
        """
        获取配置项
        
        Args:
            key: 配置键(支持点号分隔)
            default: 默认值
        
        Returns:
            配置值
        """
        keys = key.split('.')
        value = self.config
        
        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default
        
        return value

# 使用示例
config = ConfigManager('config.yaml')
db_host = config.get('database.host')
servers = config.get('servers', [])
```

### 日志记录

**结构化日志**:

```python
import logging
import json
from datetime import datetime
from typing import Any, Dict

class StructuredLogger:
    """结构化日志类"""
    
    def __init__(self, name: str, log_file: str = None):
        self.logger = logging.getLogger(name)
        self.logger.setLevel(logging.INFO)
        
        # 控制台处理器
        console_handler = logging.StreamHandler()
        console_handler.setFormatter(logging.Formatter('%(message)s'))
        self.logger.addHandler(console_handler)
        
        # 文件处理器
        if log_file:
            file_handler = logging.FileHandler(log_file)
            file_handler.setFormatter(logging.Formatter('%(message)s'))
            self.logger.addHandler(file_handler)
    
    def log(self, level: str, message: str, **kwargs) -> None:
        """
        记录日志
        
        Args:
            level: 日志级别
            message: 日志消息
            **kwargs: 额外字段
        """
        log_data = {
            'timestamp': datetime.utcnow().isoformat(),
            'level': level,
            'message': message,
            **kwargs
        }
        
        log_method = getattr(self.logger, level.lower())
        log_method(json.dumps(log_data))
    
    def info(self, message: str, **kwargs) -> None:
        self.log('INFO', message, **kwargs)
    
    def error(self, message: str, **kwargs) -> None:
        self.log('ERROR', message, **kwargs)
    
    def warning(self, message: str, **kwargs) -> None:
        self.log('WARNING', message, **kwargs)

# 使用示例
logger = StructuredLogger('my_app', log_file='app.log')
logger.info('Task started', task='backup_database', server='server1')
logger.error('Task failed', task='backup_database', error='Connection timeout')
```

### 错误处理

**异常处理装饰器**:

```python
import functools
from typing import Callable, Any

def handle_errors(
    max_retries: int = 3,
    delay: int = 1,
    exceptions: tuple = (Exception,)
):
    """
    错误处理装饰器
    
    Args:
        max_retries: 最大重试次数
        delay: 重试延迟(秒)
        exceptions: 捕获的异常类型
    """
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs) -> Any:
            import time
            
            for attempt in range(max_retries):
                try:
                    return func(*args, **kwargs)
                except exceptions as e:
                    if attempt == max_retries - 1:
                        raise
                    time.sleep(delay * (2 ** attempt))  # 指数退避
            return None
        return wrapper
    return decorator

# 使用示例
@handle_errors(max_retries=3, delay=1, exceptions=(ConnectionError, TimeoutError))
def fetch_data(url: str) -> dict:
    import requests
    response = requests.get(url, timeout=10)
    response.raise_for_status()
    return response.json()
```

## 三、自动化场景

### 服务器巡检

```python
import subprocess
import json
from typing import Dict, List

class ServerInspector:
    """服务器巡检类"""
    
    def __init__(self, servers: List[Dict]):
        self.servers = servers
    
    def check_disk_usage(self, threshold: float = 80.0) -> List[Dict]:
        """
        检查磁盘使用率
        
        Args:
            threshold: 告警阈值(%)
        
        Returns:
            异常服务器列表
        """
        alerts = []
        
        for server in self.servers:
            ssh = SSHClient(
                host=server['host'],
                username=server['username'],
                key_file=server.get('key_file')
            )
            
            try:
                returncode, stdout, stderr = ssh.execute('df -h | grep -v tmpfs | tail -n +2')
                
                for line in stdout.strip().split('\n'):
                    parts = line.split()
                    filesystem = parts[0]
                    usage = float(parts[4].rstrip('%'))
                    mount = parts[5]
                    
                    if usage > threshold:
                        alerts.append({
                            'server': server['host'],
                            'filesystem': filesystem,
                            'usage': usage,
                            'mount': mount
                        })
            finally:
                ssh.close()
        
        return alerts
    
    def check_memory_usage(self, threshold: float = 80.0) -> List[Dict]:
        """
        检查内存使用率
        
        Args:
            threshold: 告警阈值(%)
        
        Returns:
            异常服务器列表
        """
        alerts = []
        
        for server in self.servers:
            ssh = SSHClient(
                host=server['host'],
                username=server['username'],
                key_file=server.get('key_file')
            )
            
            try:
                returncode, stdout, stderr = ssh.execute('free | grep Mem')
                parts = stdout.split()
                total = int(parts[1])
                used = int(parts[2])
                usage = (used / total) * 100
                
                if usage > threshold:
                    alerts.append({
                        'server': server['host'],
                        'usage': usage,
                        'total': total,
                        'used': used
                    })
            finally:
                ssh.close()
        
        return alerts

# 使用示例
inspector = ServerInspector(config.get('servers'))
disk_alerts = inspector.check_disk_usage(threshold=80.0)
memory_alerts = inspector.check_memory_usage(threshold=85.0)
```

### 批量部署

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict

class BatchDeployer:
    """批量部署类"""
    
    def __init__(self, servers: List[Dict], max_workers: int = 10):
        self.servers = servers
        self.max_workers = max_workers
    
    def deploy(self, local_path: str, remote_path: str, command: str) -> Dict[str, Any]:
        """
        批量部署
        
        Args:
            local_path: 本地文件路径
            remote_path: 远程文件路径
            command: 部署后执行的命令
        
        Returns:
            部署结果
        """
        results = {
            'success': [],
            'failed': []
        }
        
        def deploy_to_server(server: Dict) -> Dict:
            ssh = SSHClient(
                host=server['host'],
                username=server['username'],
                key_file=server.get('key_file')
            )
            
            try:
                # 上传文件
                ssh.upload_file(local_path, remote_path)
                
                # 执行命令
                returncode, stdout, stderr = ssh.execute(command)
                
                if returncode == 0:
                    return {'server': server['host'], 'status': 'success', 'output': stdout}
                else:
                    return {'server': server['host'], 'status': 'failed', 'error': stderr}
            except Exception as e:
                return {'server': server['host'], 'status': 'failed', 'error': str(e)}
            finally:
                ssh.close()
        
        # 并发部署
        with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
            futures = {
                executor.submit(deploy_to_server, server): server
                for server in self.servers
            }
            
            for future in as_completed(futures):
                result = future.result()
                if result['status'] == 'success':
                    results['success'].append(result)
                else:
                    results['failed'].append(result)
        
        return results

# 使用示例
deployer = BatchDeployer(config.get('servers'))
result = deployer.deploy(
    local_path='/tmp/app.jar',
    remote_path='/opt/app/app.jar',
    command='systemctl restart myapp'
)
```

### 定时任务

```python
import schedule
import time
from typing import Callable

class TaskScheduler:
    """任务调度类"""
    
    def __init__(self):
        self.tasks = []
    
    def add_task(
        self,
        task_func: Callable,
        schedule_time: str,
        task_name: str = None
    ) -> None:
        """
        添加定时任务
        
        Args:
            task_func: 任务函数
            schedule_time: 调度时间(Cron 格式)
            task_name: 任务名称
        """
        # 解析 Cron 表达式(简化版)
        parts = schedule_time.split()
        if len(parts) != 5:
            raise ValueError("Invalid cron expression")
        
        minute, hour, day, month, weekday = parts
        
        # 添加任务
        if minute == '*' and hour == '*':
            schedule.every().minute.do(task_func)
        elif minute == '0' and hour != '*':
            schedule.every().day.at(f"{hour}:00").do(task_func)
        # ... 其他 Cron 表达式解析
        
        self.tasks.append({
            'name': task_name or task_func.__name__,
            'func': task_func,
            'schedule': schedule_time
        })
    
    def run(self) -> None:
        """运行调度器"""
        while True:
            schedule.run_pending()
            time.sleep(1)

# 使用示例
scheduler = TaskScheduler()

def backup_database():
    logger.info('Starting database backup')
    # 备份逻辑
    logger.info('Database backup completed')

scheduler.add_task(backup_database, '0 2 * * *', 'backup_database')
scheduler.run()
```

## 四、性能优化

### 并发处理

**多线程**:

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Callable, Any

def parallel_execute(
    func: Callable,
    items: List[Any],
    max_workers: int = 10
) -> List[Any]:
    """
    并行执行函数
    
    Args:
        func: 执行函数
        items: 参数列表
        max_workers: 最大线程数
    
    Returns:
        结果列表
    """
    results = []
    
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {executor.submit(func, item): item for item in items}
        
        for future in as_completed(futures):
            try:
                result = future.result()
                results.append(result)
            except Exception as e:
                logger.error(f"Task failed: {e}")
    
    return results

# 使用示例
def process_server(server: Dict) -> Dict:
    # 处理服务器
    return {'server': server['host'], 'status': 'success'}

results = parallel_execute(process_server, config.get('servers'), max_workers=20)
```

**多进程**:

```python
from multiprocessing import Pool
from typing import List, Callable, Any

def parallel_process(
    func: Callable,
    items: List[Any],
    processes: int = 4
) -> List[Any]:
    """
    多进程执行函数
    
    Args:
        func: 执行函数
        items: 参数列表
        processes: 进程数
    
    Returns:
        结果列表
    """
    with Pool(processes=processes) as pool:
        results = pool.map(func, items)
    
    return results
```

### 缓存优化

```python
import functools
from typing import Callable, Any
import time

def cache_result(ttl: int = 300):
    """
    结果缓存装饰器
    
    Args:
        ttl: 缓存时间(秒)
    """
    cache = {}
    
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs) -> Any:
            key = str((args, sorted(kwargs.items())))
            
            if key in cache:
                result, timestamp = cache[key]
                if time.time() - timestamp < ttl:
                    return result
            
            result = func(*args, **kwargs)
            cache[key] = (result, time.time())
            return result
        
        return wrapper
    return decorator

# 使用示例
@cache_result(ttl=60)
def get_server_info(server_ip: str) -> Dict:
    # 获取服务器信息
    return {'ip': server_ip, 'status': 'running'}
```

## 五、最佳实践

### 代码规范

**类型注解**:

```python
from typing import List, Dict, Optional, Tuple

def process_data(
    data: List[Dict[str, Any]],
    filters: Optional[Dict[str, str]] = None
) -> Tuple[int, List[Dict]]:
    """
    处理数据
    
    Args:
        data: 数据列表
        filters: 过滤条件
    
    Returns:
        (处理数量, 结果列表)
    """
    if filters:
        data = [item for item in data if all(item.get(k) == v for k, v in filters.items())]
    
    return len(data), data
```

**文档字符串**:

```python
def calculate_cost(
    instances: List[Dict],
    region: str = 'us-east-1',
    discount: float = 0.0
) -> float:
    """
    计算 EC2 实例成本
    
    Args:
        instances: 实例列表,每个实例包含 type 和 hours 字段
        region: AWS 区域
        discount: 折扣率(0-1)
    
    Returns:
        总成本(美元)
    
    Raises:
        ValueError: 如果实例列表为空或折扣率无效
    
    Examples:
        >>> instances = [{'type': 't2.micro', 'hours': 24}]
        >>> calculate_cost(instances)
        0.72
    """
    if not instances:
        raise ValueError("Instances list cannot be empty")
    
    if not 0 <= discount <= 1:
        raise ValueError("Discount must be between 0 and 1")
    
    # 成本计算逻辑
    total_cost = 0.0
    for instance in instances:
        # ...
        pass
    
    return total_cost * (1 - discount)
```

### 测试

**单元测试**:

```python
import unittest
from unittest.mock import Mock, patch

class TestServerInspector(unittest.TestCase):
    """服务器巡检测试"""
    
    def setUp(self):
        self.servers = [
            {'host': 'server1.example.com', 'username': 'user'}
        ]
        self.inspector = ServerInspector(self.servers)
    
    @patch('__main__.SSHClient')
    def test_check_disk_usage(self, mock_ssh_client):
        """测试磁盘使用率检查"""
        # Mock SSH 客户端
        mock_client = Mock()
        mock_client.execute.return_value = (
            0,
            '/dev/sda1        50G   40G   10G  80% /',
            ''
        )
        mock_ssh_client.return_value = mock_client
        
        # 执行测试
        alerts = self.inspector.check_disk_usage(threshold=75.0)
        
        # 验证结果
        self.assertEqual(len(alerts), 1)
        self.assertEqual(alerts[0]['usage'], 80.0)

if __name__ == '__main__':
    unittest.main()
```

## 小结

- **常用库**:使用 subprocess 管理进程,pathlib 处理路径,requests 发送 HTTP 请求,paramiko 执行 SSH 操作,boto3 管理 AWS 资源
- **开发模式**:使用 YAML 管理配置,结构化日志记录,装饰器处理错误,类型注解提高代码可读性
- **自动化场景**:服务器巡检监控资源使用率,批量部署并发执行,定时任务自动调度
- **性能优化**:使用多线程或多进程并发处理,缓存结果减少重复计算
- **最佳实践**:遵循代码规范,编写单元测试,使用类型注解和文档字符串

---

## 常见问题

### Q1:Python 脚本如何打包发布?

**方案一:PyInstaller**:

```bash
# 安装 PyInstaller
pip install pyinstaller

# 打包脚本
pyinstaller --onefile --name myscript myscript.py

# 生成可执行文件
# dist/myscript
```

**方案二:Setuptools**:

```python
# setup.py
from setuptools import setup, find_packages

setup(
    name='my-automation-tool',
    version='1.0.0',
    packages=find_packages(),
    install_requires=[
        'requests>=2.28.0',
        'paramiko>=3.0.0',
        'boto3>=1.26.0'
    ],
    entry_points={
        'console_scripts': [
            'mytool=mytool.cli:main'
        ]
    }
)
```

```bash
# 安装
pip install -e .

# 使用
mytool --help
```

### Q2:Python 脚本如何处理敏感信息?

**使用环境变量**:

```python
import os
from dotenv import load_dotenv

# 加载 .env 文件
load_dotenv()

# 读取环境变量
db_password = os.getenv('DB_PASSWORD')
api_key = os.getenv('API_KEY')
```

**使用 AWS Secrets Manager**:

```python
import boto3
import json

def get_secret(secret_name: str, region: str = 'us-east-1') -> dict:
    """从 AWS Secrets Manager 获取密钥"""
    client = boto3.client('secretsmanager', region_name=region)
    response = client.get_secret_value(SecretId=secret_name)
    return json.loads(response['SecretString'])

# 使用示例
db_credentials = get_secret('production/db/credentials')
```

### Q3:Python 脚本如何实现优雅退出?

**信号处理**:

```python
import signal
import sys
from typing import Callable

class GracefulExit:
    """优雅退出处理"""
    
    def __init__(self):
        self.shutdown = False
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """信号处理器"""
        logger.info(f"Received signal {signum}, shutting down...")
        self.shutdown = True
    
    def register_cleanup(self, cleanup_func: Callable) -> None:
        """注册清理函数"""
        self.cleanup_func = cleanup_func
    
    def __del__(self):
        """析构函数"""
        if hasattr(self, 'cleanup_func') and self.shutdown:
            self.cleanup_func()

# 使用示例
exit_handler = GracefulExit()

def cleanup():
    # 清理资源
    logger.info("Cleaning up resources...")

exit_handler.register_cleanup(cleanup)

while not exit_handler.shutdown:
    # 主循环
    pass
```

## 参考资源

- [Python 官方文档](https://docs.python.org/3/)
- [Python 最佳实践](https://docs.python-guide.org/)
- [boto3 文档](https://boto3.amazonaws.com/v1/documentation/api/latest/index.html)
- [paramiko 文档](http://www.paramiko.org/)
- [kubernetes-python 客户端](https://github.com/kubernetes-client/python)
