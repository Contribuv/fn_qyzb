(function() {
    const prefix = window.GATEWAY_PREFIX || '';
    let currentDomain = '';
    let currentPort = 7786;
    let isRunning = false;
    let portCheckTimer = null;
    let lastLogCount = 0;

    function init() {
        loadCerts();
        loadStatus();
        setInterval(loadStatus, 5000);
        setInterval(loadLogs, 3000);
        initPortCheck();
        initHelpToggle();
    }

    function initHelpToggle() {
        const helpToggle = document.getElementById('helpToggle');
        const helpCard = document.getElementById('helpCard');
        if (helpToggle && helpCard) {
            helpToggle.addEventListener('click', function() {
                helpCard.style.display = helpCard.style.display === 'none' ? 'block' : 'none';
            });
        }
    }

    function initPortCheck() {
        const portInput = document.getElementById('portInput');
        if (!portInput) return;

        portInput.addEventListener('input', function() {
            if (isRunning) return;
            if (portCheckTimer) {
                clearTimeout(portCheckTimer);
            }
            setPortStatus('checking', '检测中...');
            portCheckTimer = setTimeout(function() {
                const port = parseInt(portInput.value);
                if (port && port > 0 && port <= 65535) {
                    checkPort(port);
                } else {
                    setPortStatus('invalid', '端口无效');
                }
            }, 300);
        });

        setTimeout(function() {
            if (!isRunning) {
                const port = parseInt(portInput.value);
                if (port && port > 0 && port <= 65535) {
                    checkPort(port);
                }
            }
        }, 500);

        const suggestBtn1 = document.getElementById('suggestBtn1');
        const suggestBtn2 = document.getElementById('suggestBtn2');
        if (suggestBtn1) {
            suggestBtn1.addEventListener('click', function() {
                if (isRunning) return;
                const port = parseInt(this.textContent);
                portInput.value = port;
                checkPort(port);
            });
        }
        if (suggestBtn2) {
            suggestBtn2.addEventListener('click', function() {
                if (isRunning) return;
                const port = parseInt(this.textContent);
                portInput.value = port;
                checkPort(port);
            });
        }
    }

    function checkPort(port) {
        if (isRunning) {
            setPortStatus('available', '反代运行中');
            hideSuggest();
            return;
        }
        fetch(prefix + '/gateway/api/check-port?port=' + port)
            .then(r => r.json())
            .then(res => {
                if (isRunning) {
                    setPortStatus('available', '反代运行中');
                    hideSuggest();
                    return;
                }
                if (res.available) {
                    setPortStatus('available', '端口可用');
                    hideSuggest();
                } else {
                    setPortStatus('unavailable', res.message || '端口被占用');
                    if (res.suggested_port) {
                        showSuggest(res.suggested_port);
                    } else {
                        hideSuggest();
                    }
                }
            })
            .catch(err => {
                setPortStatus('unknown', '检测失败');
                hideSuggest();
            });
    }

    function setPortStatus(status, text) {
        const statusEl = document.getElementById('portStatus');
        const textEl = document.getElementById('portStatusText');
        if (!statusEl || !textEl) return;

        statusEl.className = 'port-status port-status-' + status;
        textEl.textContent = text;
    }

    function showSuggest(suggestedPort) {
        const suggestEl = document.getElementById('portSuggest');
        const btn1 = document.getElementById('suggestBtn1');
        const btn2 = document.getElementById('suggestBtn2');
        if (!suggestEl) return;

        suggestEl.style.display = 'flex';
        if (btn1) {
            btn1.textContent = suggestedPort;
            btn1.style.display = 'inline-block';
        }
        if (btn2) {
            btn2.textContent = suggestedPort + 1;
            btn2.style.display = 'inline-block';
        }
    }

    function hideSuggest() {
        const suggestEl = document.getElementById('portSuggest');
        if (suggestEl) {
            suggestEl.style.display = 'none';
        }
    }

    function loadCerts() {
        fetch(prefix + '/gateway/api/certs')
            .then(r => r.json())
            .then(res => {
                const select = document.getElementById('domainSelect');
                select.innerHTML = '';
                if (res.certs && res.certs.length > 0) {
                    res.certs.forEach(domain => {
                        const opt = document.createElement('option');
                        opt.value = domain;
                        opt.textContent = domain;
                        select.appendChild(opt);
                    });
                } else {
                    const opt = document.createElement('option');
                    opt.value = '';
                    opt.textContent = '暂无可用证书，请先在飞牛系统中申请';
                    select.appendChild(opt);
                }
            })
            .catch(err => {
                console.error('加载证书失败', err);
            });
    }

    function loadStatus() {
        fetch(prefix + '/gateway/api/status')
            .then(r => r.json())
            .then(res => {
                const wasRunning = isRunning;
                isRunning = res.running || false;
                currentDomain = res.domain || '';
                currentPort = res.port || 7786;
                updateUI();
                if (isRunning && !wasRunning) {
                    loadLogs();
                }
            })
            .catch(err => {
                console.error('加载状态失败', err);
            });
    }

    function loadLogs() {
        fetch(prefix + '/gateway/api/logs?limit=50')
            .then(r => r.json())
            .then(res => {
                if (!res.logs) return;
                const logBox = document.getElementById('logBox');
                if (!logBox) return;

                if (res.logs.length === 0 && isRunning) {
                    logBox.innerHTML = '';
                    addLog('success', '反向代理运行中');
                    return;
                }

                if (res.logs.length !== lastLogCount) {
                    lastLogCount = res.logs.length;
                    logBox.innerHTML = '';
                    res.logs.forEach(entry => {
                        const rawLevel = (entry.level || entry.Level || 'INFO').toUpperCase();
                        let type = 'info';
                        if (rawLevel === 'ERROR') type = 'error';
                        else if (rawLevel === 'SUCCESS' || rawLevel === 'WARN') type = rawLevel.toLowerCase();
                        const msg = entry.msg || entry.Msg || entry.message || '';
                        const time = entry.time || entry.Time || '';
                        appendLogLine(type, msg, time);
                    });
                    logBox.scrollTop = logBox.scrollHeight;
                }
            })
            .catch(err => {
                // 静默失败
            });
    }

    function updateUI() {
        const statusText = document.getElementById('statusText');
        const startBtn = document.getElementById('startBtn');
        const stopBtn = document.getElementById('stopBtn');
        const urlBox = document.getElementById('urlBox');
        const noUrl = document.getElementById('noUrl');
        const openBtn = document.getElementById('openBtn');
        const portInput = document.getElementById('portInput');

        if (isRunning) {
            portInput.value = currentPort;
            portInput.disabled = true;
            statusText.className = 'status-running';
            statusText.innerHTML = '<span class="status-dot"></span>运行中';
            startBtn.style.display = 'none';
            stopBtn.style.display = 'inline-flex';
            urlBox.style.display = 'flex';
            noUrl.style.display = 'none';
            openBtn.style.display = 'inline-flex';

            setPortStatus('available', '反代运行中');
            hideSuggest();

            const url = `https://${currentDomain}:${currentPort}`;
            document.getElementById('proxyUrl').textContent = url;
        } else {
            portInput.disabled = false;
            statusText.className = 'status-stopped';
            statusText.innerHTML = '<span class="status-dot"></span>未启动';
            startBtn.style.display = 'inline-flex';
            stopBtn.style.display = 'none';
            urlBox.style.display = 'none';
            noUrl.style.display = 'inline';
            openBtn.style.display = 'none';
        }
    }

    window.startProxy = function() {
        const domain = document.getElementById('domainSelect').value;
        const port = parseInt(document.getElementById('portInput').value) || 7786;

        if (!domain) {
            alert('请选择域名');
            return;
        }

        if (!port || port < 1 || port > 65535) {
            alert('请输入有效的端口号');
            return;
        }

        setPortStatus('checking', '启动中...');

        fetch(prefix + '/gateway/api/start', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ domain, port })
        })
            .then(r => r.json())
            .then(res => {
                if (res.success) {
                    addLog('info', '反向代理启动中...');
                    setTimeout(function() {
                        loadStatus();
                        loadLogs();
                    }, 1000);
                } else {
                    const msg = res.message || '未知错误';
                    alert('启动失败：' + msg);
                    addLog('error', '启动失败：' + msg);
                    setPortStatus('unavailable', msg);
                    checkPort(port);
                }
            })
            .catch(err => {
                alert('启动请求失败');
                setPortStatus('unknown', '请求失败');
            });
    };

    window.stopProxy = function() {
        if (!confirm('确定停止反向代理？')) return;

        fetch(prefix + '/gateway/api/stop', { method: 'POST' })
            .then(r => r.json())
            .then(res => {
                if (res.success) {
                    addLog('info', '反向代理已停止');
                    setTimeout(function() {
                        loadStatus();
                        loadLogs();
                        const port = parseInt(document.getElementById('portInput').value);
                        if (port) checkPort(port);
                    }, 500);
                }
            })
            .catch(err => {
                console.error('停止失败', err);
            });
    };

    window.reloadCert = function() {
        fetch(prefix + '/gateway/api/certs')
            .then(r => r.json())
            .then(res => {
                if (res.certs) {
                    addLog('success', '证书列表已刷新');
                    loadCerts();
                }
            });
    };

    window.copyUrl = function() {
        const url = document.getElementById('proxyUrl').textContent;
        navigator.clipboard.writeText(url).then(() => {
            addLog('success', '地址已复制到剪贴板');
        });
    };

    window.openAdmin = function() {
        const url = document.getElementById('proxyUrl').textContent;
        window.open(url + '/admin', '_blank');
    };

    function appendLogLine(type, msg, time) {
        const logBox = document.getElementById('logBox');
        const line = document.createElement('div');
        line.className = 'log-line log-' + type;
        if (time) {
            line.textContent = `[${time}] ${msg}`;
        } else {
            line.textContent = msg;
        }
        logBox.appendChild(line);
    }

    function addLog(type, msg) {
        const logBox = document.getElementById('logBox');
        const line = document.createElement('div');
        line.className = 'log-line log-' + type;
        const time = new Date().toLocaleTimeString();
        line.textContent = `[${time}] ${msg}`;
        logBox.appendChild(line);
        logBox.scrollTop = logBox.scrollHeight;
    }

    init();
})();
