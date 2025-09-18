// 测试改造后的API BigInt支持
const testCases = [
    {
        name: "Java Long 类型测试",
        params: '["9223372036854775807L", "123456789L"]',
        expected: "应该正确解析Java Long类型"
    },
    {
        name: "大整数测试", 
        params: '["9007199254740992", "18446744073709551615"]',
        expected: "应该正确处理超出JavaScript安全范围的大整数"
    },
    {
        name: "混合类型测试",
        params: '["123L", 456, "789.123", true, "hello"]',
        expected: "应该正确处理各种混合类型"
    },
    {
        name: "JSON对象BigInt测试",
        params: '{"id": "9223372036854775807L", "count": 100, "name": "test"}',
        expected: "应该正确处理包含BigInt的JSON对象"
    }
];

async function testAPI() {
    console.log('🧪 开始测试改造后的API BigInt支持\n');
    
    for (const testCase of testCases) {
        console.log(`📋 ${testCase.name}`);
        console.log(`📥 输入参数: ${testCase.params}`);
        console.log(`🎯 预期结果: ${testCase.expected}`);
        
        try {
            const response = await fetch('http://localhost:8080/invoke', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    interface: 'com.example.TestService',
                    method: 'testMethod',
                    params: testCase.params
                })
            });
            
            const result = await response.text();
            console.log(`✅ 响应状态: ${response.status}`);
            console.log(`📤 服务器响应: ${result.substring(0, 200)}${result.length > 200 ? '...' : ''}`);
            
        } catch (error) {
            console.log(`❌ 请求失败: ${error.message}`);
        }
        
        console.log('─'.repeat(60));
    }
    
    console.log('\n🎉 测试完成！');
    console.log('\n💡 说明:');
    console.log('- 即使Dubbo服务不存在，我们主要测试参数解析过程');
    console.log('- 关注服务器日志中的参数处理信息');
    console.log('- 检查是否正确移除L后缀并转换BigInt格式');
}

// 如果在Node.js环境中运行
if (typeof require !== 'undefined') {
    // Node.js环境需要fetch polyfill
    const fetch = require('node-fetch');
    testAPI();
} else {
    // 浏览器环境
    console.log('请在浏览器控制台中运行 testAPI() 函数');
}

// 导出测试函数供浏览器使用
if (typeof window !== 'undefined') {
    window.testAPI = testAPI;
}