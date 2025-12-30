package main

import (
	"html/template"
	"net/http"
	"time"
)

// handleSimple 处理精简版页面
func (ws *WebServer) handleSimple(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	t := template.Must(template.New("simple").Parse(simpleIndexHTML))
	data := map[string]interface{}{
		"Registry": ws.registry,
		"App":      ws.app,
		"Timeout":  ws.timeout,
		"Version":  time.Now().Unix(),
	}
	t.Execute(w, data)
}

const simpleIndexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dubbo Invoke Web UI (Simple)</title>
    <link rel="icon" type="image/png" href="/favicon.ico">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: white;
            min-height: 100vh; padding: 20px;
        }
        .container {
            max-width: 1400px; /* Wider for 2:8 layout */
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 0 10px rgba(0,0,0,0.05);
            overflow: hidden;
            width: 100%;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 25px;
            text-align: center;
            border-bottom: 1px solid #eee;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; font-weight: 300; }
        .header p { font-size: 1.1em; opacity: 0.9; }
        
        .main-content { 
            padding: 20px;
            min-height: calc(100vh - 200px);
        }
        
        /* Simple Layout */
        .simple-row {
            display: flex;
            gap: 20px;
            height: 800px;
        }
        
        .simple-left {
            flex: 0 0 30%;
            width: 30%;
            display: flex;
            flex-direction: column;
        }
        
        .simple-right {
            flex: 0 0 70%;
            width: 70%;
            display: flex;
            flex-direction: column;
        }
        
        .panel { 
            background: #fff; 
            border-radius: 8px; 
            padding: 20px; 
            border: 1px solid #e1e5e9;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            display: flex;
            flex-direction: column;
            height: 100%;
        }
        
        .panel h2 { 
            color: #333; 
            margin-bottom: 15px; 
            font-size: 1.1em; 
            font-weight: 400; 
            text-align: left;
            border-bottom: none;
            padding-left: 5px;
            display: flex;
            align-items: center;
        }
        .panel h2::before { margin-right: 8px; font-size: 1.1em; }
        
        .service-call-panel h2::before { content: '🔧'; }
        .result-panel h2::before { content: '📊'; }
        
        /* Form elements */
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; color: #555; font-size: 13px; }
        input, select, textarea {
            width: 100%; padding: 8px 10px;
            border: 1px solid #e0e0e0; border-radius: 4px;
            font-size: 13px; background-color: #fff;
        }
        input:focus, select:focus, textarea:focus { outline: none; border-color: #4a90e2; }
        textarea { resize: vertical; min-height: 80px; font-family: monospace; }
        
        .btn {
            background: #4a90e2; color: white; border: none;
            padding: 8px 16px; border-radius: 4px; cursor: pointer;
            font-size: 13px; transition: background 0.2s ease;
            margin-right: 10px; margin-bottom: 10px;
        }
        .btn:hover { background: #3a7dca; }
        .btn-secondary { background: #6c6fe2; }
        .btn-secondary:hover { background: #5a5dca; }
        
        /* JSON Tree */
        .json-tree-item { margin: 0; padding: 0; }
        .json-tree-toggle {
            display: inline-block; width: 12px; height: 12px; margin-right: 4px;
            cursor: pointer; user-select: none; font-size: 10px; line-height: 12px;
            text-align: center; color: #666; border: 1px solid #ccc;
            background: #fff; border-radius: 2px;
        }
        .json-tree-toggle.collapsed::before { content: '▶'; }
        .json-tree-toggle.expanded::before { content: '▼'; }
        .json-tree-key { color: #0066cc; font-weight: bold; }
        .json-tree-string { color: #008000; }
        .json-tree-number { color: #ff6600; }
        .json-tree-boolean { color: #cc0066; font-style: italic; }
        .json-tree-null { color: #999; font-style: italic; }
        .json-tree-children { margin-left: 20px; border-left: 1px dotted #ccc; padding-left: 10px; }
        .json-tree-children.collapsed { display: none; }
        
        .result {
            flex: 1; overflow: auto; padding: 10px;
            background: #f8f9fa; border: 1px solid #e0e0e0; border-radius: 4px;
            font-family: monospace; font-size: 13px; white-space: pre-wrap;
        }
        .success { border-color: #4caf50; background-color: #f1f8e9; }
        .error { border-color: #ff5252; background-color: #ffebee; color: #d32f2f; }
        
        .loading { 
            display: none; text-align: center; padding: 25px; 
            color: #5c6bc0; font-weight: 500;
            background-color: rgba(92, 107, 192, 0.05); border-radius: 8px;
        }
        .spinner {
            border: 3px solid rgba(92, 107, 192, 0.1); border-top: 3px solid #5c6bc0;
            border-radius: 50%; width: 30px; height: 30px;
            animation: spin 1s linear infinite; margin: 0 auto 10px;
        }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
        
        .result-actions { display: flex; gap: 8px; align-items: center; margin-left: auto; }
        .icon-btn {
            background: none; border: none; cursor: pointer; padding: 6px;
            border-radius: 4px; font-size: 16px;
        }
        .icon-btn:hover { background-color: #f0f0f0; }
        
        /* Responsive */
        @media (max-width: 1024px) {
            .simple-row { flex-direction: column; height: auto; }
            .simple-left, .simple-right { width: 100%; flex: none; }
            .simple-left { margin-bottom: 20px; }
            .panel { min-height: 400px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Dubbo Invoke Web UI (精简版)</h1>
            <p>专注服务调用与结果展示</p>
        </div>
        <div class="main-content">
            <div class="simple-row">
                <!-- Left: Service Call (20%) -->
                <div class="simple-left">
                    <div class="panel service-call-panel">
                        <h2>服务调用</h2>
                        <div class="form-group">
                            <label for="callFormat">调用格式:</label>
                            <select id="callFormat" onchange="toggleCallFormat()">
                                <option value="traditional">传统格式</option>
                                <option value="expression" selected>表达式格式</option>
                            </select>
                        </div>
                        
                        <!-- Expression Format -->
                        <div id="expressionFormat" style="display: none;">
                            <div class="form-group">
                                <label>注册中心配置:</label>
                                <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 10px;">
                                    <select id="registryTypeExpr" onchange="onRegistryTypeChangeExpr()" style="width: 120px; flex-shrink: 0;">
                                        <option value="zookeeper">ZooKeeper</option>
                                        <option value="nacos">Nacos</option>
                                        <option value="dubbo">Dubbo</option>
                                    </select>
                                    <div id="zkEnvironmentContainerExpr" style="display: none; flex: 1;">
                                        <select id="zkEnvironmentExpr" onchange="onZkEnvironmentChangeExpr()" style="width: 100%;">
                                            <option value="">请选择环境</option>
                                            <option value="dev">开发环境 (dev)</option>
                                            <option value="uat">用户验收测试 (uat)</option>
                                            <option value="tat">技术验收测试 (tat)</option>
                                            <option value="fat">功能验收测试 (fat)</option>
                                            <option value="pre">预生产环境 (pre)</option>
                                            <option value="prod">生产环境 (prod)</option>
                                        </select>
                                    </div>
                                    <input type="text" id="registryAddressExpr" value="{{.Registry}}" placeholder="127.0.0.1:2181" style="flex: 1;" readonly>
                                    <button class="btn btn-secondary" onclick="testConnection()" style="margin: 0; white-space: nowrap;">🔗 测试</button>
                                </div>
                            </div>
                            <div class="form-group" id="namespaceGroupExpr" style="display: block;">
                                <label for="namespaceExpr">命名空间:</label>
                                <input type="text" id="namespaceExpr" placeholder="public" value="public">
                            </div>
                            <div class="form-group">
                                <label for="expression">调用表达式:</label>
                                <textarea id="expression" placeholder='invoke com.example.UserService.getUserById(123)'>invoke com.example.UserService.getUserById(123)</textarea>
                            </div>
                        </div>
                        
                        <!-- Traditional Format -->
                        <div id="traditionalFormat">
                            <div class="form-group">
                                <label>注册中心配置:</label>
                                <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 10px;">
                                    <select id="registryType" onchange="onRegistryTypeChange()" style="width: 120px; flex-shrink: 0;">
                                        <option value="zookeeper">ZooKeeper</option>
                                        <option value="nacos">Nacos</option>
                                        <option value="dubbo">Dubbo</option>
                                    </select>
                                    <div id="zkEnvironmentContainer" style="display: none; flex: 1;">
                                        <select id="zkEnvironment" onchange="onZkEnvironmentChange()" style="width: 100%;">
                                            <option value="">请选择环境</option>
                                            <option value="dev">开发环境 (dev)</option>
                                            <option value="uat">用户验收测试 (uat)</option>
                                            <option value="tat">技术验收测试 (tat)</option>
                                            <option value="fat">功能验收测试 (fat)</option>
                                            <option value="pre">预生产环境 (pre)</option>
                                            <option value="prod">生产环境 (prod)</option>
                                        </select>
                                    </div>
                                    <input type="text" id="registryAddress" placeholder="127.0.0.1:2181" value="127.0.0.1:2181" style="flex: 1;" readonly>
                                    <button class="btn btn-secondary" onclick="testConnection()" style="margin: 0; white-space: nowrap;">🔗 测试</button>
                                </div>
                            </div>
                            <div class="form-group" id="namespaceGroup" style="display: none;">
                                <label for="namespace">命名空间:</label>
                                <input type="text" id="namespace" placeholder="public" value="public">
                            </div>
                            <div class="form-group">
                                <label for="serviceName">服务名:</label>
                                <input type="text" id="serviceName" placeholder="com.example.UserService" value="com.example.UserService">
                            </div>
                            <div class="form-group">
                                <label for="methodName">方法名:</label>
                                <input type="text" id="methodName" placeholder="getUserById" value="getUserById">
                            </div>
                            <div class="form-group">
                                <label for="parameters">参数 (JSON数组):</label>
                                <textarea id="parameters" placeholder='[123]'>[123]</textarea>
                            </div>
                        </div>
                        
                        <div id="traditionalTypes" class="form-group">
                            <label for="types">参数类型 (可选):</label>
                            <input type="text" id="types" placeholder="java.lang.Long">
                        </div>
                        
                        <div class="btn-group" style="margin-top: auto;">
                            <button class="btn" onclick="invokeService()">🚀 调用</button>
                        </div>
                        <!-- Hidden elements for compatibility -->
                        <div id="serviceList" style="display:none;"></div>
                    </div>
                </div>
                
                <!-- Right: Result (80%) -->
                <div class="simple-right">
                    <div class="panel result-panel">
                        <h2>
                            <span>调用结果</span>
                            <div class="result-actions">
                                <button class="icon-btn compress" onclick="toggleJsonFormat()" title="压缩/美化">🗜️</button>
                                <button class="icon-btn line-numbers" onclick="toggleLineNumbers()" title="行号">🔢</button>
                                <button class="icon-btn expand-all" onclick="toggleExpandAll()" title="展开/收缩">📂</button>
                                <button class="icon-btn copy" onclick="copyResult()" title="复制">📋</button>
                                <button class="icon-btn save" onclick="saveResult()" title="保存">💾</button>
                                <button class="icon-btn clear" onclick="clearResult()" title="清空">🗑️</button>
                            </div>
                        </h2>
                        <div id="loading" class="loading">
                            <div class="spinner"></div>
                            正在调用服务...
                        </div>
                        <div id="result" class="result" style="display: none;"></div>
                    </div>
                </div>
            </div>
        </div>
        <div class="footer" style="text-align: center; margin-top: 20px; padding: 20px; color: #666; font-size: 14px;">
            <a href="/full" style="color: #4a90e2; text-decoration: none; margin: 0 10px;">体验完整版</a>
            <span style="color: #ccc;">|</span>
            <a href="https://github.com/dubbo/dubbo-go" target="_blank" style="color: #666; text-decoration: none; margin: 0 10px;">Dubbo-Go</a>
        </div>
    </div>
    
    <script>
        // Copy constants and functions from indexHTML
        const ZOOKEEPER_ENVIRONMENTS = {
            dev: { name: '开发环境', address: '10.7.8.40:2181', servicePath: 'dubbo' },
            uat: { name: '用户验收测试', address: '10.7.8.42:2181', servicePath: 'uat' },
            tat: { name: '技术验收测试', address: '10.6.12.153:2181', servicePath: 'tat' },
            fat: { name: '功能验收测试', address: '10.6.12.205:2181', servicePath: 'fat' },
            pre: { name: '预生产环境', address: 'mse-4ec83a20-zk.mse.aliyuncs.com:2181', servicePath: 'pre' },
            prod: { name: '生产环境', address: 'mse-2cd54c90-zk.mse.aliyuncs.com:2181', servicePath: 'prod' }
        };
        
        let originalJsonData = null;
        let isJsonCompressed = false;
        let showLineNumbers = false;
        let isAllExpanded = true;
        
        // ... Core functions (toggleCallFormat, onRegistryTypeChange, onZkEnvironmentChange, etc.) ...
        // I'll inline the essential logic here to ensure it works
        
        function toggleCallFormat() {
            const format = document.getElementById('callFormat').value;
            const traditional = document.getElementById('traditionalFormat');
            const expression = document.getElementById('expressionFormat');
            const traditionalTypes = document.getElementById('traditionalTypes');
            
            if (format === 'expression') {
                traditional.style.display = 'none';
                expression.style.display = 'block';
                traditionalTypes.style.display = 'none';
                
                // 同步配置：从传统模式 -> 表达式模式
                const registryType = document.getElementById('registryType').value;
                const registryAddress = document.getElementById('registryAddress').value;
                const namespace = document.getElementById('namespace').value;
                
                if (document.getElementById('registryTypeExpr')) document.getElementById('registryTypeExpr').value = registryType;
                if (document.getElementById('registryAddressExpr')) document.getElementById('registryAddressExpr').value = registryAddress;
                if (document.getElementById('namespaceExpr')) document.getElementById('namespaceExpr').value = namespace;
                
                onRegistryTypeChangeExpr();
            } else {
                traditional.style.display = 'block';
                expression.style.display = 'none';
                traditionalTypes.style.display = 'block';
                
                // 同步配置：从表达式模式 -> 传统模式
                const registryType = document.getElementById('registryTypeExpr').value;
                const registryAddress = document.getElementById('registryAddressExpr').value;
                const namespace = document.getElementById('namespaceExpr').value;
                
                if (document.getElementById('registryType')) document.getElementById('registryType').value = registryType;
                if (document.getElementById('registryAddress')) document.getElementById('registryAddress').value = registryAddress;
                if (document.getElementById('namespace')) document.getElementById('namespace').value = namespace;
                
                onRegistryTypeChange();
            }
        }
        
        function onRegistryTypeChange() {
            const type = document.getElementById('registryType').value;
            const zkEnvironmentContainer = document.getElementById('zkEnvironmentContainer');
            const registryAddress = document.getElementById('registryAddress');
            
            document.getElementById('namespaceGroup').style.display = type === 'nacos' ? 'block' : 'none';
            
            if (type === 'zookeeper') {
                zkEnvironmentContainer.style.display = 'block';
                registryAddress.style.display = 'none';
                registryAddress.readOnly = true;
                registryAddress.placeholder = '请先选择环境';
                registryAddress.value = '';
                document.getElementById('zkEnvironment').value = '';
            } else {
                zkEnvironmentContainer.style.display = 'none';
                registryAddress.style.display = 'block';
                registryAddress.readOnly = false;
                
                if (type === 'nacos') {
                    registryAddress.placeholder = '127.0.0.1:8848';
                    if (!registryAddress.value || registryAddress.value === '127.0.0.1:2181' || registryAddress.value === '127.0.0.1:8080') {
                        registryAddress.value = '127.0.0.1:8848';
                    }
                } else if (type === 'dubbo') {
                    registryAddress.placeholder = '127.0.0.1:8080';
                    if (!registryAddress.value || registryAddress.value === '127.0.0.1:2181' || registryAddress.value === '127.0.0.1:8848') {
                        registryAddress.value = '127.0.0.1:8080';
                    }
                }
            }
        }
        
        function onRegistryTypeChangeExpr() {
            const type = document.getElementById('registryTypeExpr').value;
            const zkEnvironmentContainer = document.getElementById('zkEnvironmentContainerExpr');
            const registryAddress = document.getElementById('registryAddressExpr');
            
            document.getElementById('namespaceGroupExpr').style.display = type === 'nacos' ? 'block' : 'none';
            
            if (type === 'zookeeper') {
                zkEnvironmentContainer.style.display = 'block';
                registryAddress.style.display = 'none';
                registryAddress.readOnly = true;
                registryAddress.placeholder = '请先选择环境';
                registryAddress.value = '';
                document.getElementById('zkEnvironmentExpr').value = '';
            } else {
                zkEnvironmentContainer.style.display = 'none';
                registryAddress.style.display = 'block';
                registryAddress.readOnly = false;
                
                if (type === 'nacos') {
                    registryAddress.placeholder = '127.0.0.1:8848';
                    if (!registryAddress.value || registryAddress.value === '127.0.0.1:2181' || registryAddress.value === '127.0.0.1:8080') {
                        registryAddress.value = '127.0.0.1:8848';
                    }
                } else if (type === 'dubbo') {
                    registryAddress.placeholder = '127.0.0.1:8080';
                    if (!registryAddress.value || registryAddress.value === '127.0.0.1:2181' || registryAddress.value === '127.0.0.1:8848') {
                        registryAddress.value = '127.0.0.1:8080';
                    }
                }
            }
        }
        
        function onZkEnvironmentChange() {
            const env = document.getElementById('zkEnvironment').value;
            const addressInput = document.getElementById('registryAddress');
            const namespaceInput = document.getElementById('namespace');
            
            if (env && ZOOKEEPER_ENVIRONMENTS[env]) {
                addressInput.value = ZOOKEEPER_ENVIRONMENTS[env].address;
                if(namespaceInput) namespaceInput.value = ZOOKEEPER_ENVIRONMENTS[env].servicePath;
            } else {
                addressInput.value = '';
                if(namespaceInput) namespaceInput.value = 'public';
            }
        }
        
        function onZkEnvironmentChangeExpr() {
            const env = document.getElementById('zkEnvironmentExpr').value;
            const addressInput = document.getElementById('registryAddressExpr');
            const namespaceInput = document.getElementById('namespaceExpr');
            
            if (env && ZOOKEEPER_ENVIRONMENTS[env]) {
                addressInput.value = ZOOKEEPER_ENVIRONMENTS[env].address;
                if(namespaceInput) namespaceInput.value = ZOOKEEPER_ENVIRONMENTS[env].servicePath;
            } else {
                addressInput.value = '';
                if(namespaceInput) namespaceInput.value = 'public';
            }
        }
        
        // Include JSONBig logic
        function parseExpression(expr) {
            const parenIndex = expr.indexOf('(');
            if (parenIndex === -1) return null;
            const methodPart = expr.substring(0, parenIndex);
            const lastDotIndex = methodPart.lastIndexOf('.');
            if (lastDotIndex === -1) return null;
            const serviceName = methodPart.substring(0, lastDotIndex);
            const methodName = methodPart.substring(lastDotIndex + 1);
            let paramsPart = expr.substring(parenIndex + 1);
            if (paramsPart.endsWith(')')) {
                paramsPart = paramsPart.substring(0, paramsPart.length - 1);
            }
            let parameters = [];
            if (paramsPart.trim()) {
                // 先应用removeLSuffix处理，确保L后缀被正确移除
                const processedParamsPart = removeLSuffix(paramsPart.trim());
                
                try {
                    // 首先尝试将整个参数部分作为JSON数组解析（只有当它是完整的JSON数组格式时）
                    if (processedParamsPart.startsWith('[') && processedParamsPart.endsWith(']')) {
                        try {
                            // 将整体数组视为单个参数（List），避免被拆分为多个独立参数
                            parameters = [JSONBig.parse(processedParamsPart)];
                        } catch (e) {
                            // 如果解析失败，说明不是有效的JSON数组，使用参数分割逻辑
                            throw e;
                        }
                    } else {
                        // 不是完整的JSON数组格式，使用参数分割逻辑
                        throw new Error('Not a complete JSON array');
                    }
                } catch (e) {
                    // 使用智能参数分割逻辑
                    const paramStrings = parseParametersFromExpression(processedParamsPart);
                    parameters = paramStrings.map(paramStr => {
                        try {
                            return JSONBig.parse(paramStr);
                        } catch (e) {
                            // 如果不是JSON格式，去除引号后返回字符串
                            if (paramStr.startsWith('"') && paramStr.endsWith('"')) {
                                return paramStr.substring(1, paramStr.length - 1);
                            }
                            // 尝试解析为数字
                            if (!isNaN(paramStr) && !isNaN(parseFloat(paramStr))) {
                                return parseFloat(paramStr);
                            }
                            return paramStr;
                        }
                    });
                }
            }
            return { serviceName, methodName, parameters };
        }
        
        // 从表达式中解析参数的函数，与后端逻辑保持一致
        function parseParametersFromExpression(paramsPart) {
            if (!paramsPart || paramsPart.trim() === '') {
                return [];
            }
            
            const params = [];
            let current = '';
            let braceCount = 0;
            let bracketCount = 0;
            let inQuotes = false;
            let escapeNext = false;
            
            for (let i = 0; i < paramsPart.length; i++) {
                const char = paramsPart[i];
                
                if (escapeNext) {
                    current += char;
                    escapeNext = false;
                    continue;
                }
                
                if (char === '\\') {
                    escapeNext = true;
                    current += char;
                    continue;
                }
                
                if (char === '"') {
                    inQuotes = !inQuotes;
                }
                
                if (!inQuotes) {
                    if (char === '{') {
                        braceCount++;
                    } else if (char === '}') {
                        braceCount--;
                    } else if (char === '[') {
                        bracketCount++;
                    } else if (char === ']') {
                        bracketCount--;
                    } else if (char === ',' && braceCount === 0 && bracketCount === 0) {
                        // 找到参数分隔符
                        const param = current.trim();
                        if (param !== '') {
                            params.push(param);
                        }
                        current = '';
                        continue;
                    }
                }
                
                current += char;
            }
            
            // 添加最后一个参数
            const param = current.trim();
            if (param !== '') {
                params.push(param);
            }
            
            return params;
        }

        // JSON-BigInt 库的简化实现，用于处理大整数
        const JSONBig = {
            parse: function(text, reviver) {
                // 首先移除Java long字面量的L后缀
                let cleanText = text.replace(/(\d+(?:\.\d+)?)L([^a-zA-Z0-9]|$)/g, '$1$2');
                
                // 仅在非字符串上下文中将超过15位的整数包装为字符串
                (function() {
                    let s = cleanText;
                    let res = '';
                    let inQuotes = false;
                    let escapeNext = false;
                    let i = 0;
                    while (i < s.length) {
                        const ch = s[i];
                        if (escapeNext) {
                            res += ch;
                            escapeNext = false;
                            i++;
                            continue;
                        }
                        if (ch === '\\') {
                            res += ch;
                            escapeNext = true;
                            i++;
                            continue;
                        }
                        if (ch === '"') {
                            res += ch;
                            inQuotes = !inQuotes;
                            i++;
                            continue;
                        }
                        if (!inQuotes && (ch === '-' || (ch >= '0' && ch <= '9'))) {
                            let j = i;
                            if (ch === '-') j++;
                            while (j < s.length && s[j] >= '0' && s[j] <= '9') j++;
                            const num = s.slice(i, j);
                            let hasL = false;
                            if (j < s.length && s[j] === 'L') {
                                hasL = true;
                                j++;
                            }
                            const prev = res.length ? res[res.length - 1] : '';
                            const next = j < s.length ? s[j] : '';
                            const prevWord = /[A-Za-z0-9_]/.test(prev);
                            const nextWord = /[A-Za-z0-9_.]/.test(next);
                            if (num.length >= 16 && !prevWord && !nextWord) {
                                res += '"' + num + '"';
                            } else {
                                res += s.slice(i, j);
                            }
                            i = j;
                            continue;
                        }
                        res += ch;
                        i++;
                    }
                    cleanText = res;
                })();
                
                try {
                    return JSON.parse(cleanText, function(key, value) {
                        // 检查是否为大整数字符串（被我们转换的）
                        if (typeof value === 'string' && /^-?\d{16,}$/.test(value)) {
                            // 超过15位的整数，保持为字符串避免精度丢失
                            return value;
                        }
                        
                        // 检查是否为超出安全范围的数字
                        if (typeof value === 'number') {
                            if (value > Number.MAX_SAFE_INTEGER || value < Number.MIN_SAFE_INTEGER) {
                                return value.toString();
                            }
                        }
                        
                        return reviver ? reviver(key, value) : value;
                    });
                } catch (error) {
                    // 降级处理：如果解析失败，尝试原始字符串
                    console.warn('JSON解析失败，使用原始字符串:', error);
                    return cleanText;
                }
            },
            
            stringify: function(value, replacer, space) {
                return JSON.stringify(value, function(key, val) {
                    // 处理大整数，确保序列化时保持精度
                    if (typeof val === 'bigint') {
                        return val.toString();
                    }
                    return replacer ? replacer(key, val) : val;
                }, space);
            }
        };
        
        // 兼容性函数：保持向后兼容
        function removeLSuffix(jsonStr) {
            return jsonStr.replace(/(\d+(?:\.\d+)?)L([^a-zA-Z0-9]|$)/g, '$1$2');
        }
        
        // JSON Tree and Result display logic
        // 处理对象中的大整数，确保它们以字符串形式显示
        function processLargeIntegers(obj) {
            if (obj === null || obj === undefined) {
                return obj;
            }
            
            if (typeof obj === 'object' && !Array.isArray(obj)) {
                // 处理对象
                const result = {};
                for (const key in obj) {
                    if (obj.hasOwnProperty(key)) {
                        result[key] = processLargeIntegers(obj[key]);
                    }
                }
                return result;
            } else if (Array.isArray(obj)) {
                // 处理数组
                return obj.map(item => processLargeIntegers(item));
            } else if (typeof obj === 'number') {
                // 处理数字，检查是否为大整数
                // 检查是否超过JavaScript安全整数范围
                if (obj > Number.MAX_SAFE_INTEGER || obj < Number.MIN_SAFE_INTEGER) {
                    return obj.toString();
                }
                // 处理15位及以上的整数（即使在安全范围内也可能有精度问题）
                if ((obj >= 1000000000000000 && obj <= Number.MAX_SAFE_INTEGER) || 
                    (obj <= -1000000000000000 && obj >= Number.MIN_SAFE_INTEGER)) {
                    return obj.toString();
                }
                return obj;
            } else if (typeof obj === 'string') {
                // 如果是空字符串，直接返回，不进行数字转换
                if (obj === '') {
                    return obj;
                }
                // 尝试将字符串转换为数字，如果转换后超过安全范围，则保持为字符串
                const num = Number(obj);
                if (!isNaN(num)) {
                    // 检查是否超过JavaScript安全整数范围
                    if (num > Number.MAX_SAFE_INTEGER || num < Number.MIN_SAFE_INTEGER) {
                        return obj; // 保持为字符串
                    }
                    // 处理15位及以上的整数
                    if ((num >= 1000000000000000 && num <= Number.MAX_SAFE_INTEGER) || 
                        (num <= -1000000000000000 && num >= Number.MIN_SAFE_INTEGER)) {
                        return obj; // 保持为字符串
                    }
                    return num; // 转换为数字
                }
                return obj;
            }
            
            return obj;
        }

        function createJsonTree(data, key = null, isRoot = true) {
            const container = document.createElement('div');
            container.className = 'json-tree-item';
            
            if (data === null) {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-null">null</span>';
                return container;
            }
            
            if (typeof data === 'string') {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-string">"' + escapeHtml(data) + '"</span>';
                return container;
            }
            
            if (typeof data === 'number') {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-number">' + data + '</span>';
                return container;
            }
            
            if (typeof data === 'boolean') {
                container.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-boolean">' + data + '</span>';
                return container;
            }
            
            const isArray = Array.isArray(data);
            const entries = isArray ? data.map((item, index) => [index, item]) : Object.entries(data);
            const isEmpty = entries.length === 0;
            
            const itemLine = document.createElement('div');
            itemLine.className = 'json-tree-item-line';
            
            if (isEmpty) {
                itemLine.innerHTML = (key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '') + '<span class="json-tree-bracket">' + (isArray ? '[]' : '{}') + '</span>';
                container.appendChild(itemLine);
                return container;
            }
            
            const toggle = document.createElement('span');
            toggle.className = 'json-tree-toggle expanded';
            
            const keySpan = key ? '<span class="json-tree-key">"' + key + '"</span>: ' : '';
            const openBracket = '<span class="json-tree-bracket">' + (isArray ? '[' : '{') + '</span>';
            const summary = '<span class="json-tree-summary">' + entries.length + ' ' + (isArray ? 'items' : 'properties') + '</span>';
            
            itemLine.innerHTML = keySpan + openBracket + summary;
            itemLine.insertBefore(toggle, itemLine.firstChild);
            
            const childrenContainer = document.createElement('div');
            childrenContainer.className = 'json-tree-children';
            
            entries.forEach(([childKey, childValue], index) => {
                const childElement = isArray ? 
                    createJsonTree(childValue, null, false) : 
                    createJsonTree(childValue, childKey, false);
                if (index < entries.length - 1) {
                    const comma = document.createElement('span');
                    comma.innerHTML = ',';
                    comma.style.color = '#666';
                    childElement.appendChild(comma);
                }
                childrenContainer.appendChild(childElement);
            });
            
            const closeLine = document.createElement('div');
            closeLine.className = 'json-tree-item-line';
            closeLine.innerHTML = '<span class="json-tree-bracket">' + (isArray ? ']' : '}') + '</span>';
            childrenContainer.appendChild(closeLine);
            
            toggle.addEventListener('click', function() {
                if (toggle.classList.contains('expanded')) {
                    toggle.classList.remove('expanded');
                    toggle.classList.add('collapsed');
                    childrenContainer.classList.add('collapsed');
                    itemLine.innerHTML = keySpan + openBracket + '<span class="json-tree-bracket">' + (isArray ? '...]' : '...}') + '</span>' + summary;
                    itemLine.insertBefore(toggle, itemLine.firstChild);
                } else {
                    toggle.classList.remove('collapsed');
                    toggle.classList.add('expanded');
                    childrenContainer.classList.remove('collapsed');
                    itemLine.innerHTML = keySpan + openBracket + summary;
                    itemLine.insertBefore(toggle, itemLine.firstChild);
                }
            });
            
            container.appendChild(itemLine);
            container.appendChild(childrenContainer);
            
            return container;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function renderJsonTree(data, container) {
            container.innerHTML = '';
            container.className = 'result json-tree';
            
            // 存储原始JSON数据用于复制功能
            originalJsonData = data;
            
            try {
                const tree = createJsonTree(data);
                container.appendChild(tree);
            } catch (error) {
                container.className = 'result';
                container.textContent = 'JSON树形展示错误: ' + error.message + '\n\n原始数据:\n' + JSON.stringify(data, null, 2);
                // 错误情况下也要存储数据
                originalJsonData = data;
            }
        }

        function displayResult(data) {
            const result = document.getElementById('result');
            result.className = 'result ' + (data.success ? 'success' : 'error');
            result.style.display = 'block';
            
            // 如果是成功调用，显示data字段的内容；如果是失败，显示error信息
            if (data.success && data.data !== undefined) {
                // 格式化显示数据，提供优雅的输出格式
                if (typeof data.data === 'string') {
                    try {
                        // 如果是JSON字符串，尝试解析并格式化
                        const parsed = JSON.parse(data.data, function(key, value) {
                            // 检查是否为大整数（超过JavaScript安全整数范围）
                            if (typeof value === 'number' && (value > Number.MAX_SAFE_INTEGER || value < Number.MIN_SAFE_INTEGER)) {
                                return value.toString();
                            }
                            // 处理19位及以上的整数
                            if (typeof value === 'number' && value >= 1000000000000000) {
                                return value.toString();
                            }
                            return value;
                        });
                        // 使用JSON树形展示
                        renderJsonTree(parsed, result);
                    } catch (e) {
                        // 如果不是JSON字符串，直接显示
                        result.className = 'result';
                        result.textContent = data.data;
                        originalJsonData = data.data;
                    }
                } else if (typeof data.data === 'object' && data.data !== null) {
                    // 如果是对象或数组，使用JSON树形展示，并处理其中的大整数
                    const processedData = processLargeIntegers(data.data);
                    renderJsonTree(processedData, result);
                } else {
                    // 如果是基础数据类型（数字、布尔值、null等），直接显示
                    result.className = 'result';
                    result.textContent = String(data.data);
                    originalJsonData = data.data;
                }
            } else if (!data.success && data.error) {
                result.className = 'result';
                result.textContent = data.error;
                originalJsonData = data.error;
            } else {
                // 兼容旧格式或其他情况
                renderJsonTree(data, result);
            }
            
            // 更新结果面板标题的状态指示器
            const resultPanelTitle = document.querySelector('.result-panel h2');
            if (resultPanelTitle) {
                const statusIndicator = data.success ? 
                    '<span style="color: #4caf50; margin-left: 8px;">●</span>' : 
                    '<span style="color: #f44336; margin-left: 8px;">●</span>';
                const statusText = data.success ? '调用成功' : '调用失败';
                
                // 构建耗时信息
                let timeInfo = '';
                if (data.totalTime) {
                    timeInfo += ' (总耗时: ' + data.totalTime + 'ms';
                    if (data.duration) {
                        timeInfo += ', 后端: ' + data.duration + 'ms';
                    }
                    timeInfo += ')';
                } else if (data.duration) {
                    timeInfo += ' (后端耗时: ' + data.duration + 'ms)';
                }
                
                // 保留复制按钮，只更新标题文本
                const titleSpan = resultPanelTitle.querySelector('span');
                if (titleSpan) {
                    titleSpan.innerHTML = '调用结果 - ' + statusText + timeInfo + statusIndicator;
                } else {
                    // 如果没有找到span，创建一个并保留原有结构
                    const actionsDiv = resultPanelTitle.querySelector('.result-actions');
                    resultPanelTitle.innerHTML = '<span>调用结果 - ' + statusText + timeInfo + statusIndicator + '</span>';
                    if (actionsDiv) {
                        resultPanelTitle.appendChild(actionsDiv);
                    }
                }
            }
        }

        function showLoading(show) {
            document.getElementById('loading').style.display = show ? 'block' : 'none';
            document.getElementById('result').style.display = show ? 'none' : 'block';
        }

        // Invoke Service
        function invokeService() {
            const format = document.getElementById('callFormat').value;
            let serviceName, methodName, parameters;
            if (format === 'expression') {
                let expr = document.getElementById('expression').value.trim();
                if (!expr) { alert('请输入调用表达式'); return; }
                // 如果表达式以"invoke "开头，去除该前缀
                if (expr.startsWith('invoke ')) {
                    expr = expr.substring(7); // 去除"invoke "前缀
                }
                const parsed = parseExpression(expr);
                if (!parsed) { alert('无效的表达式格式'); return; }
                serviceName = parsed.serviceName;
                methodName = parsed.methodName;
                parameters = parsed.parameters;
            } else {
                serviceName = document.getElementById('serviceName').value.trim();
                methodName = document.getElementById('methodName').value.trim();
                const paramsText = document.getElementById('parameters').value.trim();
                if (!serviceName || !methodName) { alert('请输入服务名和方法名'); return; }
                try {
                    // 使用JSONBig解析参数，支持大整数和Java long类型
                    parameters = paramsText ? JSONBig.parse(paramsText) : [];
                } catch (e) { alert('参数格式错误，请使用JSON数组格式: ' + e.message); return; }
            }
            // 获取参数类型信息
            let types = '';
            if (format === 'traditional') {
                types = document.getElementById('types').value.trim();
            } else {
                // 表达式格式：根据参数自动推断类型
                if (parameters && Array.isArray(parameters) && parameters.length > 0) {
                    types = parameters.map(param => {
                        if (typeof param === 'string') {
                            return 'java.lang.String';
                        } else if (typeof param === 'number') {
                            if (Number.isInteger(param)) {
                                // 检查是否超出Integer范围，如果超出则使用Long
                                if (param > 2147483647 || param < -2147483648) {
                                    return 'java.lang.Long';
                                } else {
                                    return 'java.lang.Integer';
                                }
                            } else {
                                    return 'java.lang.Double';
                            }
                        } else if (typeof param === 'boolean') {
                            return 'java.lang.Boolean';
                        } else if (Array.isArray(param)) {
                            return 'java.util.List';
                        } else if (typeof param === 'object' && param !== null) {
                            return 'java.lang.Object';
                        } else {
                            return 'java.lang.Object';
                        }
                    }).join(',');
                }
            }
            
            // 获取注册中心类型和地址
            let registryType, registryAddress;
            if (format === 'expression') {
                registryType = document.getElementById('registryTypeExpr').value;
                registryAddress = document.getElementById('registryAddressExpr').value.trim();
            } else {
                registryType = document.getElementById('registryType').value;
                registryAddress = document.getElementById('registryAddress').value.trim();
            }
            
            if (!registryAddress) {
                alert('请先输入注册中心地址');
                return;
            }
            
            // 根据注册中心类型构建完整的registry地址
            let registry;
            if (registryType === 'zookeeper') {
                registry = 'zookeeper://' + registryAddress;
            } else if (registryType === 'nacos') {
                registry = 'nacos://' + registryAddress;
            } else if (registryType === 'dubbo') {
                registry = 'dubbo://' + registryAddress;
            } else {
                registry = registryAddress;
            }
            
            let namespace;
            if (format === 'expression') {
                const namespaceExprElement = document.getElementById('namespaceExpr');
                namespace = namespaceExprElement ? namespaceExprElement.value.trim() : 'public';
            } else {
                const namespaceElement = document.getElementById('namespace');
                namespace = namespaceElement ? namespaceElement.value.trim() : 'public';
            }
            const request = {
                serviceName: serviceName, methodName: methodName,
                parameters: parameters,
                types: types ? types.split(',').map(t => t.trim()) : [],
                registry: registry, app: '{{.App}}', timeout: 10000,
                namespace: namespace
            };
            showLoading(true);
            const startTime = Date.now(); // 记录前端调用开始时间
            fetch('/api/invoke', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSONBig.stringify(request)
            })
            .then(response => {
                if (response.ok) {
                    return response.json();
                } else {
                    // 对于错误响应，直接返回文本内容
                    return response.text().then(text => ({
                        success: false,
                        error: text
                    }));
                }
            })
            .then(data => { 
                showLoading(false); 
                const totalTime = Date.now() - startTime; // 计算总耗时
                data.totalTime = totalTime; // 添加总耗时到响应数据
                displayResult(data); 
            })
            .catch(error => {
                showLoading(false);
                const totalTime = Date.now() - startTime;
                displayResult({ success: false, error: '网络错误: ' + error.message, totalTime: totalTime });
            });
        }
        
        // ... Other helper functions ...
        function testConnection() {
            const format = document.getElementById('callFormat').value;
            let registryType, registryAddress;
            
            if (format === 'expression') {
                registryType = document.getElementById('registryTypeExpr').value;
                registryAddress = document.getElementById('registryAddressExpr').value.trim();
            } else {
                registryType = document.getElementById('registryType').value;
                registryAddress = document.getElementById('registryAddress').value.trim();
            }
            
            if (!registryAddress) {
                alert('请先输入注册中心地址');
                return;
            }
            
            let registry;
            if (registryType === 'zookeeper') {
                registry = 'zookeeper://' + registryAddress;
            } else if (registryType === 'nacos') {
                registry = 'nacos://' + registryAddress;
            } else if (registryType === 'dubbo') {
                registry = 'dubbo://' + registryAddress;
            } else {
                registry = registryAddress;
            }
            
            let namespace;
            if (format === 'expression') {
                const namespaceExprElement = document.getElementById('namespaceExpr');
                namespace = namespaceExprElement ? namespaceExprElement.value.trim() : 'public';
            } else {
                const namespaceElement = document.getElementById('namespace');
                namespace = namespaceElement ? namespaceElement.value.trim() : 'public';
            }
            
            // 获取按钮元素
            const btn = window.event ? window.event.target : null;
            let originalText = '';
            if (btn) {
                originalText = btn.textContent;
                btn.textContent = '测试中...';
                btn.disabled = true;
            }
            
            fetch('/api/check-connection', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    registry: registry,
                    namespace: namespace,
                    app: '{{.App}}'
                })
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    alert('✅ 连通性测试通过');
                } else {
                    alert('❌ 连通性测试失败，网络不可达\n详细原因：' + (data.error || '未知错误'));
                }
            })
            .catch(e => {
                alert('❌ 连通性测试失败，网络不可达\n详细原因：' + e.message);
            })
            .finally(() => {
                if (btn) {
                    btn.textContent = originalText;
                    btn.disabled = false;
                }
            });
        }
        function generateExample() { alert('生成示例 (简易版)'); }
        
        function copyResult() {
            // 优先使用存储的原始JSON数据
            let textToCopy = '';
            
            if (originalJsonData !== null) {
                // 使用存储的原始JSON数据，格式化输出
                textToCopy = JSON.stringify(originalJsonData, null, 2);
            } else {
                // 如果没有存储的数据，回退到使用元素的文本内容
                const resultElement = document.getElementById('result');
                if (!resultElement) {
                    alert('暂无结果数据可复制');
                    return;
                }
                
                // 如果是树形展示，提示无法复制
                if (resultElement.classList.contains('json-tree')) {
                    alert('暂无结果数据可复制');
                    return;
                } else {
                    textToCopy = resultElement.textContent.trim();
                }
            }
            
            if (!textToCopy) {
                alert('暂无结果数据可复制');
                return;
            }
            
            // 创建临时文本区域用于复制
            const textarea = document.createElement('textarea');
            textarea.value = textToCopy;
            document.body.appendChild(textarea);
            textarea.select();
            
            try {
                document.execCommand('copy');
                alert('结果已复制到剪贴板');
            } catch (err) {
                // 如果复制失败，提供下载选项
                const blob = new Blob([textToCopy], { type: 'application/json' });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'dubbo-invoke-result-' + new Date().toISOString().slice(0,19).replace(/:/g, '-') + '.json';
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
                alert('复制失败，已自动下载结果文件');
            } finally {
                document.body.removeChild(textarea);
            }
        }

        // 压缩/美化JSON显示切换
        function toggleJsonFormat() {
            if (!originalJsonData) {
                alert('暂无JSON数据');
                return;
            }
            
            const resultElement = document.getElementById('result');
            isJsonCompressed = !isJsonCompressed;
            
            if (isJsonCompressed) {
                // 压缩显示
                resultElement.className = 'result';
                resultElement.textContent = JSON.stringify(originalJsonData);
            } else {
                // 美化显示（树形展示）
                renderJsonTree(originalJsonData, resultElement);
            }
            
            // 更新按钮图标
            const btn = document.querySelector('.compress');
            btn.innerHTML = isJsonCompressed ? '📄' : '🗜️';
            btn.title = isJsonCompressed ? '美化JSON' : '压缩JSON';
        }

        // 显示/隐藏行号
        function toggleLineNumbers() {
            const resultElement = document.getElementById('result');
            showLineNumbers = !showLineNumbers;
            
            if (showLineNumbers) {
                resultElement.classList.add('show-line-numbers');
                // 如果是JSON树形展示，转换为格式化的纯文本显示
                if (resultElement.classList.contains('json-tree') && originalJsonData) {
                    const formattedJson = JSON.stringify(originalJsonData, null, 2);
                    resultElement.className = 'result show-line-numbers';
                    addLineNumbers(resultElement, formattedJson);
                } else {
                    // 对于纯文本内容，直接添加行号
                    addLineNumbers(resultElement);
                }
            } else {
                resultElement.classList.remove('show-line-numbers');
                // 恢复原始显示模式
                if (originalJsonData && typeof originalJsonData === 'object') {
                    // 恢复JSON树形展示
                    renderJsonTree(originalJsonData, resultElement);
                } else {
                    // 移除行号，恢复原始内容
                    removeLineNumbers(resultElement);
                }
            }
            
            // 更新按钮图标
            const btn = document.querySelector('.line-numbers');
            btn.innerHTML = showLineNumbers ? '🔢' : '🔢';
            btn.title = showLineNumbers ? '隐藏行号' : '显示行号';
        }

        // 添加行号到结果内容
        function addLineNumbers(element, content = null) {
            const textContent = content || element.textContent || element.innerText;
            const lines = textContent.split('\n');
            
            // 清空元素并重新构建带行号的内容
            element.innerHTML = '';
            lines.forEach((line, index) => {
                const lineDiv = document.createElement('div');
                lineDiv.className = 'line';
                lineDiv.textContent = line || ' '; // 空行显示空格
                element.appendChild(lineDiv);
            });
        }

        // 移除行号，恢复原始内容
        function removeLineNumbers(element) {
            const lines = element.querySelectorAll('.line');
            if (lines.length > 0) {
                const content = Array.from(lines).map(line => line.textContent).join('\n');
                element.textContent = content;
            }
        }

        // 全部展开/收缩
        function toggleExpandAll() {
            const resultElement = document.getElementById('result');
            if (!resultElement.classList.contains('json-tree')) {
                alert('当前不是树形展示模式');
                return;
            }
            
            isAllExpanded = !isAllExpanded;
            const toggles = resultElement.querySelectorAll('.json-tree-toggle');
            const children = resultElement.querySelectorAll('.json-tree-children');
            
            toggles.forEach((toggle, index) => {
                if (isAllExpanded) {
                    toggle.classList.remove('collapsed');
                    toggle.classList.add('expanded');
                    if (children[index]) {
                        children[index].classList.remove('collapsed');
                    }
                } else {
                    toggle.classList.remove('expanded');
                    toggle.classList.add('collapsed');
                    if (children[index]) {
                        children[index].classList.add('collapsed');
                    }
                }
            });
            
            // 更新按钮图标
            const btn = document.querySelector('.expand-all');
            btn.innerHTML = isAllExpanded ? '📁' : '📂';
            btn.title = isAllExpanded ? '全部收缩' : '全部展开';
        }

        // 保存结果
        function saveResult() {
            if (!originalJsonData) {
                alert('暂无结果数据可保存');
                return;
            }
            
            const jsonString = JSON.stringify(originalJsonData, null, 2);
            const blob = new Blob([jsonString], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'dubbo-invoke-result-' + new Date().toISOString().slice(0,19).replace(/:/g, '-') + '.json';
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }

        // 清空结果
        function clearResult() {
            const resultElement = document.getElementById('result');
            resultElement.innerHTML = '';
            resultElement.className = 'result';
            resultElement.style.display = 'none';
            originalJsonData = null;
            
            // 重置状态
            isJsonCompressed = false;
            showLineNumbers = false;
            isAllExpanded = true;
            
            // 重置按钮状态
            const compressBtn = document.querySelector('.compress');
            const lineNumbersBtn = document.querySelector('.line-numbers');
            const expandBtn = document.querySelector('.expand-all');
            
            if (compressBtn) {
                compressBtn.innerHTML = '🗜️';
                compressBtn.title = '压缩JSON';
            }
            if (lineNumbersBtn) {
                lineNumbersBtn.innerHTML = '🔢';
                lineNumbersBtn.title = '显示行号';
            }
            if (expandBtn) {
                expandBtn.innerHTML = '📂';
                expandBtn.title = '全部展开/收缩';
            }
            
            // 隐藏结果面板标题的状态指示器
            const resultPanelTitle = document.querySelector('.result-panel h2');
            if (resultPanelTitle) {
                const titleSpan = resultPanelTitle.querySelector('span');
                if (titleSpan) {
                    titleSpan.innerHTML = '调用结果';
                }
            }
        }

        window.onload = function() {
            toggleCallFormat();
            onRegistryTypeChangeExpr();
        };
    </script>
</body>
</html>`
