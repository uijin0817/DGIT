const { contextBridge, ipcRenderer } = require('electron');

// 안전한 IPC API를 window.electron에 노출
contextBridge.exposeInMainWorld('electron', {
    // ====== 앱 컨트롤 ======
    
    // 트래픽 라이트 버튼 (macOS 윈도우 컨트롤)
    closeApp: () => ipcRenderer.send('close-app'),
    minimizeApp: () => ipcRenderer.send('minimize-app'),
    maximizeApp: () => ipcRenderer.send('maximize-app'),
    
    // 개발자 도구 및 앱 관리
    toggleDevTools: () => ipcRenderer.send('toggle-dev-tools'),
    restartApp: () => ipcRenderer.send('restart-app'),
    
    // ====== 파일 시스템 및 다이얼로그 ======
    
    // 폴더/파일 선택 다이얼로그
    selectFolder: () => ipcRenderer.invoke('select-folder'),
    selectFile: (options) => ipcRenderer.invoke('select-file', options),
    
    // 파일 읽기/쓰기
    readFile: (path) => ipcRenderer.invoke('read-file', path),
    writeFile: (path, data) => ipcRenderer.invoke('write-file', path, data),
    
    // 디렉토리 스캔
    scanDirectory: (path) => ipcRenderer.invoke('scan-directory', path),
    
    // ====== DGit CLI 연동 ======
    
    // 기본 DGit 명령
    dgit: {
        // 일반 명령 실행
        command: (command, args, projectPath) => 
            ipcRenderer.invoke('dgit-command', command, args, projectPath),
        
        // 상태 확인
        status: (projectPath) => 
            ipcRenderer.invoke('dgit-status', projectPath),
        
        // 커밋
        commit: (projectPath, message) => 
            ipcRenderer.invoke('dgit-commit', projectPath, message),
        
        // 저장소 초기화
        init: (projectPath) => 
            ipcRenderer.invoke('dgit-init', projectPath),
        
        // 로그 조회
        log: (projectPath, limit) => 
            ipcRenderer.invoke('dgit-log', projectPath, limit),
        
        // 파일 추가
        add: (projectPath, files) => 
            ipcRenderer.invoke('dgit-command', 'add', files || ['.'], projectPath),
        
        // 브랜치 관련
        createBranch: (projectPath, branchName) => 
            ipcRenderer.invoke('dgit-command', 'checkout', ['-b', branchName], projectPath),
        
        switchBranch: (projectPath, branchName) => 
            ipcRenderer.invoke('dgit-command', 'checkout', [branchName], projectPath),
        
        listBranches: (projectPath) => 
            ipcRenderer.invoke('dgit-command', 'branch', [], projectPath),
        
          // 복원 - ⭐⭐ 수정: DGit restore 명령어 사용
          restore: (projectPath, versionOrHash, files) => 
            ipcRenderer.invoke('dgit-restore', projectPath, versionOrHash, files),
        
        // 파일 상세 분석 (dgit show)
        showFile: (projectPath, fileName) => 
            ipcRenderer.invoke('dgit-show-file', projectPath, fileName),
        
        // Diff 보기
        diff: (projectPath, commitHash1, commitHash2) => 
            ipcRenderer.invoke('dgit-command', 'diff', [commitHash1, commitHash2], projectPath)
    },
    
    // ====== 설정 관리 ======
    
    // 앱 설정
    config: {
        load: () => ipcRenderer.invoke('load-config'),
        save: (config) => ipcRenderer.invoke('save-config', config)
    },
    
    // 최근 프로젝트
    recentProjects: {
        load: () => ipcRenderer.invoke('load-recent-projects'),
        save: (project) => ipcRenderer.invoke('save-recent-project', project)
    },
    
    // ====== 시스템 정보 ======
    
    // 앱 정보
    getAppInfo: () => ipcRenderer.invoke('get-app-info'),
    
    // 시스템 알림
    showNotification: (title, body) => ipcRenderer.invoke('show-notification', title, body),
    
    // Finder에서 보기
    showInFinder: (path) => ipcRenderer.invoke('show-in-finder', path),
    
    // ====== 이벤트 리스너 ======
    
    // 이벤트 리스너 등록/해제
    on: (channel, callback) => {
        const validChannels = [
            'window-maximized',
            'window-unmaximized',
            'window-focus',
            'window-blur',
            'app-update-available',
            'dgit-status-changed'
        ];
        
        if (validChannels.includes(channel)) {
            ipcRenderer.on(channel, callback);
            
            // 정리 함수 반환
            return () => ipcRenderer.removeListener(channel, callback);
        }
    },
    
    // 일회성 이벤트 리스너
    once: (channel, callback) => {
        const validChannels = [
            'window-maximized',
            'window-unmaximized', 
            'window-focus',
            'window-blur',
            'app-update-available',
            'dgit-status-changed'
        ];
        
        if (validChannels.includes(channel)) {
            ipcRenderer.once(channel, callback);
        }
    },
    
    // ====== 유틸리티 ======
    
    // 플랫폼 정보
    platform: process.platform,
    
    // dgit 객체 안에 추가
showFile: (projectPath, fileName) => 
    ipcRenderer.invoke('dgit-show-file', projectPath, fileName),
    // 환경 변수
    env: {
        NODE_ENV: process.env.NODE_ENV,
        isDevelopment: process.env.NODE_ENV === 'development'
    },
    
    // 경로 유틸리티
    path: {
        join: (...args) => require('path').join(...args),
        basename: (path, ext) => require('path').basename(path, ext),
        dirname: (path) => require('path').dirname(path),
        extname: (path) => require('path').extname(path)
    },
    
    // 파일 시스템 유틸리티
    fs: {
        exists: async (path) => {
            try {
                const result = await ipcRenderer.invoke('read-file', path);
                return result.success;
            } catch {
                return false;
            }
        },
        
        isDirectory: async (path) => {
            try {
                const result = await ipcRenderer.invoke('scan-directory', path);
                return result.success;
            } catch {
                return false;
            }
        }
    },
    
    // ====== 보안 관련 ======
    
    // 안전한 HTML 삽입을 위한 sanitize 함수
    sanitize: {
        html: (html) => {
            const div = document.createElement('div');
            div.textContent = html;
            return div.innerHTML;
        },
        
        attribute: (attr) => {
            return attr.replace(/[<>"'&]/g, (match) => {
                const escapes = {
                    '<': '&lt;',
                    '>': '&gt;',
                    '"': '&quot;',
                    "'": '&#x27;',
                    '&': '&amp;'
                };
                return escapes[match];
            });
        }
    },
    
    // 로그 출력
    log: {
        info: (message, ...args) => console.log('[INFO]', message, ...args),
        warn: (message, ...args) => console.warn('[WARN]', message, ...args),
        error: (message, ...args) => console.error('[ERROR]', message, ...args),
        debug: (message, ...args) => {
            if (process.env.NODE_ENV === 'development') {
                console.log('[DEBUG]', message, ...args);
            }
        }
    }
});

// 보안을 위해 Node.js API 직접 접근 차단
contextBridge.exposeInMainWorld('versions', {
    node: () => process.versions.node,
    chrome: () => process.versions.chrome,
    electron: () => process.versions.electron,
    v8: () => process.versions.v8
});

// 개발 모드에서만 추가 디버깅 기능 제공
if (process.env.NODE_ENV === 'development') {
    contextBridge.exposeInMainWorld('debug', {
        // IPC 통신 로그
        ipcLog: (channel, data) => {
            console.log(`[IPC] ${channel}:`, data);
        },
        
        // 메모리 사용량
        memory: () => process.memoryUsage(),
        
        // 성능 측정
        performance: {
            mark: (name) => performance.mark(name),
            measure: (name, startMark, endMark) => performance.measure(name, startMark, endMark),
            getEntriesByType: (type) => performance.getEntriesByType(type)
        }
    });
}