// DOM元素引用
const registryTypeSelect = document.getElementById('registryType');
const zkEnvironmentSelect = document.getElementById('zkEnvironment');
const zkEnvSection = document.getElementById('zookeeperEnvSection');
const otherRegistrySection = document.getElementById('otherRegistrySection');
const serverAddressInput = document.getElementById('serverAddress');
const servicePathInput = document.getElementById('servicePath');
const customAddressInput = document.getElementById('customAddress');
const customPathInput = document.getElementById('customPath');
const configInfoDiv = document.getElementById('configInfo');
const connectionStatusDiv = document.getElementById('connectionStatus');

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    loadSavedConfiguration();
    updateConfigDisplay();
});

// 处理注册中心类型变更
function handleRegistryTypeChange() {
    const selectedType = registryTypeSelect.value;
    currentConfig.registryType = selectedType;
    
    // 重置相关字段
    resetConfigFields();
    
    if (selectedType === 'zookeeper') {
        // 显示Zookeeper环境选择区域
        zkEnvSection.style.display = 'block';
        otherRegistrySection.style.display = 'none';
        
        // 清空并禁用通用输入框
        serverAddressInput.readOnly = true;
        servicePathInput.readOnly = true;
        serverAddressInput.placeholder = '请先选择环境';
        servicePathInput.placeholder = '请先选择环境';
    } else if (selectedType === 'nacos' || selectedType === 'dubbo') {
        // 隐藏Zookeeper环境选择，显示其他注册中心配置
        zkEnvSection.style.display = 'none';
        otherRegistrySection.style.display = 'block';
        
        // 启用通用输入框
        serverAddressInput.readOnly = false;
        servicePathInput.readOnly = false;
        serverAddressInput.placeholder = `请输入${REGISTRY_TYPES[selectedType].name}服务器地址`;
        servicePathInput.placeholder = `请输入${REGISTRY_TYPES[selectedType].name}服务路径`;
        
        // 设置默认端口提示
        customAddressInput.placeholder = `例如: localhost:${REGISTRY_TYPES[selectedType].defaultPort}`;
    } else {
        // 未选择注册中心类型
        zkEnvSection.style.display = 'none';
        otherRegistrySection.style.display = 'none';
        serverAddressInput.readOnly = true;
        servicePathInput.readOnly = true;
        serverAddressInput.placeholder = '请先选择注册中心类型';
        servicePathInput.placeholder = '请先选择注册中心类型';
    }
    
    updateConfigDisplay();
    updateConnectionStatus('offline', '未连接');
}

// 处理Zookeeper环境变更
function handleZkEnvironmentChange() {
    const selectedEnv = zkEnvironmentSelect.value;
    
    if (selectedEnv && ZOOKEEPER_ENVIRONMENTS[selectedEnv]) {
        const envConfig = ZOOKEEPER_ENVIRONMENTS[selectedEnv];
        
        // 更新当前配置
        currentConfig.environment = selectedEnv;
        currentConfig.address = envConfig.address;
        currentConfig.servicePath = envConfig.servicePath;
        
        // 更新输入框显示
        serverAddressInput.value = envConfig.address;
        servicePathInput.value = envConfig.servicePath;
        
        showAlert('success', `已选择${envConfig.name}环境配置`);
    } else {
        // 清空配置
        currentConfig.environment = '';
        currentConfig.address = '';
        currentConfig.servicePath = '';
        serverAddressInput.value = '';
        servicePathInput.value = '';
    }
    
    updateConfigDisplay();
    updateConnectionStatus('offline', '未连接');
}

// 测试连接
async function testConnection() {
    if (!validateConfiguration()) {
        return;
    }
    
    updateConnectionStatus('testing', '正在测试连接...');
    
    try {
        // 模拟连接测试（实际项目中这里应该调用后端API）
        await simulateConnectionTest();
        
        currentConfig.isConnected = true;
        updateConnectionStatus('online', '连接成功');
        showAlert('success', '注册中心连接测试成功！');
        
    } catch (error) {
        currentConfig.isConnected = false;
        updateConnectionStatus('offline', '连接失败');
        showAlert('error', `连接测试失败: ${error.message}`);
    }
}

// 模拟连接测试
function simulateConnectionTest() {
    return new Promise((resolve, reject) => {
        setTimeout(() => {
            // 模拟连接成功/失败的逻辑
            const success = Math.random() > 0.3; // 70%成功率
            if (success) {
                resolve();
            } else {
                reject(new Error('无法连接到注册中心服务器'));
            }
        }, 2000);
    });
}

// 保存配置
function saveConfiguration() {
    if (!validateConfiguration()) {
        return;
    }
    
    try {
        // 更新当前配置
        if (currentConfig.registryType !== 'zookeeper') {
            currentConfig.address = customAddressInput.value || serverAddressInput.value;
            currentConfig.servicePath = customPathInput.value || servicePathInput.value;
        }
        
        // 保存到本地存储
        localStorage.setItem('registryConfig', JSON.stringify(currentConfig));
        
        showAlert('success', '配置已保存成功！');
        updateConfigDisplay();
        
    } catch (error) {
        showAlert('error', `保存配置失败: ${error.message}`);
    }
}

// 重置表单
function resetForm() {
    // 重置选择框
    registryTypeSelect.value = '';
    zkEnvironmentSelect.value = '';
    
    // 重置输入框
    serverAddressInput.value = '';
    servicePathInput.value = '';
    customAddressInput.value = '';
    customPathInput.value = '';
    
    // 重置当前配置
    currentConfig = {
        registryType: '',
        environment: '',
        address: '',
        servicePath: '',
        isConnected: false
    };
    
    // 隐藏所有动态区域
    zkEnvSection.style.display = 'none';
    otherRegistrySection.style.display = 'none';
    
    // 更新显示
    updateConfigDisplay();
    updateConnectionStatus('offline', '未连接');
    
    showAlert('warning', '表单已重置');
}

// 验证配置
function validateConfiguration() {
    if (!currentConfig.registryType) {
        showAlert('error', '请选择注册中心类型');
        return false;
    }
    
    if (currentConfig.registryType === 'zookeeper') {
        if (!currentConfig.environment) {
            showAlert('error', '请选择Zookeeper环境');
            return false;
        }
    } else {
        const address = customAddressInput.value || serverAddressInput.value;
        const path = customPathInput.value || servicePathInput.value;
        
        if (!address) {
            showAlert('error', '请输入服务器地址');
            return false;
        }
        
        if (!path) {
            showAlert('error', '请输入服务路径');
            return false;
        }
    }
    
    return true;
}

// 更新配置显示
function updateConfigDisplay() {
    let configHtml = '';
    
    if (currentConfig.registryType) {
        configHtml += `<p><strong>注册中心类型:</strong> ${REGISTRY_TYPES[currentConfig.registryType].name}</p>`;
        
        if (currentConfig.registryType === 'zookeeper' && currentConfig.environment) {
            const envConfig = ZOOKEEPER_ENVIRONMENTS[currentConfig.environment];
            configHtml += `<p><strong>环境:</strong> ${envConfig.name} (${currentConfig.environment})</p>`;
        }
        
        if (currentConfig.address) {
            configHtml += `<p><strong>服务器地址:</strong> ${currentConfig.address}</p>`;
        }
        
        if (currentConfig.servicePath) {
            configHtml += `<p><strong>服务路径:</strong> ${currentConfig.servicePath}</p>`;
        }
        
        if (currentConfig.isConnected) {
            configHtml += `<p><strong>连接状态:</strong> <span style="color: #27ae60;">已连接</span></p>`;
        }
    } else {
        configHtml = '<p>请选择注册中心类型和环境</p>';
    }
    
    configInfoDiv.innerHTML = configHtml;
}

// 更新连接状态
function updateConnectionStatus(status, message) {
    const statusIndicator = connectionStatusDiv.querySelector('.status-indicator');
    const statusText = connectionStatusDiv.querySelector('span:last-child');
    
    // 移除所有状态类
    statusIndicator.classList.remove('online', 'offline', 'testing');
    
    // 添加新状态类
    statusIndicator.classList.add(status);
    statusText.textContent = message;
}

// 显示提示信息
function showAlert(type, message) {
    // 移除现有的提示
    const existingAlert = document.querySelector('.alert');
    if (existingAlert) {
        existingAlert.remove();
    }
    
    // 创建新提示
    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;
    
    // 插入到表单区域后面
    const formSection = document.querySelector('.form-section');
    formSection.parentNode.insertBefore(alert, formSection.nextSibling);
    
    // 3秒后自动移除
    setTimeout(() => {
        if (alert.parentNode) {
            alert.remove();
        }
    }, 3000);
}

// 加载已保存的配置
function loadSavedConfiguration() {
    try {
        const savedConfig = localStorage.getItem('registryConfig');
        if (savedConfig) {
            const config = JSON.parse(savedConfig);
            
            // 恢复注册中心类型
            if (config.registryType) {
                registryTypeSelect.value = config.registryType;
                currentConfig.registryType = config.registryType;
                handleRegistryTypeChange();
                
                // 恢复Zookeeper环境选择
                if (config.registryType === 'zookeeper' && config.environment) {
                    zkEnvironmentSelect.value = config.environment;
                    handleZkEnvironmentChange();
                } else if (config.registryType !== 'zookeeper') {
                    // 恢复其他注册中心的配置
                    customAddressInput.value = config.address || '';
                    customPathInput.value = config.servicePath || '';
                    serverAddressInput.value = config.address || '';
                    servicePathInput.value = config.servicePath || '';
                    currentConfig.address = config.address || '';
                    currentConfig.servicePath = config.servicePath || '';
                }
            }
            
            showAlert('success', '已加载保存的配置');
        }
    } catch (error) {
        console.error('加载配置失败:', error);
        showAlert('warning', '加载保存的配置失败，使用默认配置');
    }
}

// 重置配置字段
function resetConfigFields() {
    currentConfig.environment = '';
    currentConfig.address = '';
    currentConfig.servicePath = '';
    currentConfig.isConnected = false;
    
    zkEnvironmentSelect.value = '';
    serverAddressInput.value = '';
    servicePathInput.value = '';
    customAddressInput.value = '';
    customPathInput.value = '';
}

// 键盘快捷键支持
document.addEventListener('keydown', function(event) {
    // Ctrl+S 保存配置
    if (event.ctrlKey && event.key === 's') {
        event.preventDefault();
        saveConfiguration();
    }
    
    // Ctrl+R 重置表单
    if (event.ctrlKey && event.key === 'r') {
        event.preventDefault();
        resetForm();
    }
    
    // Ctrl+T 测试连接
    if (event.ctrlKey && event.key === 't') {
        event.preventDefault();
        testConnection();
    }
});