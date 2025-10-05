const { app, BrowserWindow, ipcMain, dialog } = require('electron');
const path = require('path');
const fs = require('fs');
const DGitIntegration = require('./dgit-integration');

// DGit 통합 인스턴스 생성
const dgit = new DGitIntegration();

// 메인 윈도우 변수
let mainWindow;

// macOS에서 앱이 활성화될 때 창 다시 생성
app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
        createMainWindow();
    }
});

// 모든 창이 닫힐 때 앱 종료 (macOS 제외)
app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') {
        app.quit();
    }
});

// 앱이 준비되면 메인 창 생성
app.whenReady().then(() => {
    createMainWindow();
});

// 메인 창 생성 함수
function createMainWindow() {
    mainWindow = new BrowserWindow({
        width: 1200,
        height: 800,
        minWidth: 800,
        minHeight: 600,
        titleBarStyle: 'hiddenInset', // macOS 스타일 타이틀바 숨김
        vibrancy: 'under-window', // macOS 투명 효과
        webPreferences: {
            nodeIntegration: false,
            contextIsolation: true,
            enableRemoteModule: false,
            preload: path.join(__dirname, 'preload.js'),
            webSecurity: true
        },
        show: false, // 로딩 완료 후 표시
        backgroundColor: '#1e1e1e', // 다크 테마 기본 배경
        icon: process.platform === 'darwin' 
            ? path.join(__dirname, 'assets', 'icon.icns')
            : path.join(__dirname, 'assets', 'icon.png')
    });

    // 로딩 완료 후 창 표시
    mainWindow.once('ready-to-show', () => {
        mainWindow.show();
        
        // 개발 모드에서 개발자 도구 열기
        if (process.argv.includes('--dev')) {
            mainWindow.webContents.openDevTools();
        }
    });

    // HTML 파일 로드
    mainWindow.loadFile(path.join(__dirname, 'renderer', 'index.html'));

    // 창 이벤트 핸들러
    mainWindow.on('closed', () => {
        mainWindow = null;
    });

    // 메뉴바 숨기기 (Windows/Linux)
    if (process.platform !== 'darwin') {
        mainWindow.setMenuBarVisibility(false);
    }
}

// ====== IPC 핸들러들 ======

// 트래픽 라이트 버튼 (macOS 윈도우 컨트롤)
ipcMain.on('close-app', () => {
    if (mainWindow) {
        mainWindow.close();
    }
});

ipcMain.on('minimize-app', () => {
    if (mainWindow) {
        mainWindow.minimize();
    }
});

ipcMain.on('maximize-app', () => {
    if (mainWindow) {
        if (mainWindow.isMaximized()) {
            mainWindow.unmaximize();
        } else {
            mainWindow.maximize();
        }
    }
});

// 폴더/파일 선택 다이얼로그
ipcMain.handle('select-folder', async () => {
    const result = await dialog.showOpenDialog(mainWindow, {
        properties: ['openDirectory'],
        title: '프로젝트 폴더 선택',
        buttonLabel: '선택'
    });
    
    if (!result.canceled && result.filePaths.length > 0) {
        return {
            success: true,
            path: result.filePaths[0],
            name: path.basename(result.filePaths[0])
        };
    }
    
    return { success: false };
});

ipcMain.handle('select-file', async (event, options = {}) => {
    const result = await dialog.showOpenDialog(mainWindow, {
        properties: ['openFile'],
        filters: options.filters || [
            { name: 'Design Files', extensions: ['psd', 'ai', 'sketch', 'fig', 'xd'] },
            { name: 'Images', extensions: ['png', 'jpg', 'jpeg', 'gif', 'svg'] },
            { name: 'All Files', extensions: ['*'] }
        ],
        title: options.title || '파일 선택',
        buttonLabel: options.buttonLabel || '선택'
    });
    
    if (!result.canceled && result.filePaths.length > 0) {
        return {
            success: true,
            path: result.filePaths[0],
            name: path.basename(result.filePaths[0])
        };
    }
    
    return { success: false };
});

// ====== DGit CLI 연동 ======

// DGit 명령 실행
ipcMain.handle('dgit-command', async (event, command, args = [], projectPath = null) => {
    try {
        const result = await dgit.executeCommand(command, args, projectPath);
        return {
            success: true,
            output: result.output,
            code: result.code
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message,
            code: error.code || -1
        };
    }
});

// DGit 상태 확인
ipcMain.handle('dgit-status', async (event, projectPath) => {
    try {
        const result = await dgit.status(projectPath);
        return {
            success: true,
            output: result.output
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message
        };
    }
});

// DGit 커밋
ipcMain.handle('dgit-commit', async (event, projectPath, message) => {
    try {
        // 먼저 모든 파일 추가
        await dgit.add(projectPath);
        
        // 그 다음 커밋
        const result = await dgit.commit(projectPath, message);
        return {
            success: true,
            output: result.output
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message
        };
    }
});

// DGit 저장소 초기화
ipcMain.handle('dgit-init', async (event, projectPath) => {
    try {
        const result = await dgit.init(projectPath);
        return {
            success: true,
            output: result.output
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message
        };
    }
});

// DGit 로그 조회
ipcMain.handle('dgit-log', async (event, projectPath, limit = 10) => {
    try {
        const result = await dgit.log(projectPath, limit);
        return {
            success: true,
            output: result.output
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message
        };
    }
});

// DGit 파일 복원
ipcMain.handle('dgit-restore', async (event, projectPath, versionOrHash, files = []) => {
    try {
        const args = [versionOrHash];
        
        // 파일 목록이 있으면 추가
        if (files && files.length > 0) {
            args.push(...files);
        }
        
        const result = await dgit.executeCommand('restore', args, projectPath);
        return {
            success: true,
            output: result.output
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message
        };
    }
});
// ====== 파일 시스템 조작 ======

// 파일 읽기
ipcMain.handle('read-file', async (event, filePath) => {
    try {
        const data = await fs.promises.readFile(filePath, 'utf8');
        return {
            success: true,
            data: data
        };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
});

// 파일 쓰기
ipcMain.handle('write-file', async (event, filePath, data) => {
    try {
        await fs.promises.writeFile(filePath, data, 'utf8');
        return { success: true };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
});

// ⭐⭐ 새로 추가: 파일 상세 분석 (dgit show)
ipcMain.handle('dgit-show-file', async (event, projectPath, fileName) => {
    try {
        const result = await dgit.executeCommand('show', [fileName], projectPath);
        return {
            success: true,
            output: result.output
        };
    } catch (error) {
        return {
            success: false,
            error: error.error || error.message
        };
    }
});

// 디렉토리 스캔
ipcMain.handle('scan-directory', async (event, dirPath) => {
    try {
        const items = await fs.promises.readdir(dirPath, { withFileTypes: true });
        const files = [];
        
        for (const item of items) {
            if (item.isFile()) {
                const filePath = path.join(dirPath, item.name);
                const stats = await fs.promises.stat(filePath);
                const ext = path.extname(item.name).toLowerCase().slice(1);
                
                // 디자인 파일들만 필터링
                const designExts = ['psd', 'ai', 'sketch', 'fig', 'xd', 'png', 'jpg', 'jpeg', 'gif', 'svg'];
                
                if (designExts.includes(ext)) {
                    files.push({
                        name: item.name,
                        path: filePath,
                        size: stats.size,
                        modified: stats.mtime,
                        type: ext
                    });
                }
            }
        }
        
        return {
            success: true,
            files: files
        };
    } catch (error) {
        console.error('===== 스캔 실패 원인 =====');
        console.error('요청된 경로:', dirPath); // 어떤 경로에서 에러가 났는지 확인
        console.error('발생한 에러:', error);   // 터미널에 전체 에러 객체 출력
        console.error('=========================');
        return {
            success: false,
            error: error.message
        };
    }
});

// ====== 설정 관리 ======

// 설정 파일 경로
const configPath = path.join(app.getPath('userData'), 'config.json');
const recentProjectsPath = path.join(app.getPath('userData'), 'recent-projects.json');

// 설정 로드
ipcMain.handle('load-config', async () => {
    try {
        if (fs.existsSync(configPath)) {
            const data = await fs.promises.readFile(configPath, 'utf8');
            return {
                success: true,
                config: JSON.parse(data)
            };
        }
        
        // 기본 설정
        const defaultConfig = {
            theme: 'dark',
            notifications: true,
            autoSave: true
        };
        
        return {
            success: true,
            config: defaultConfig
        };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
});

// 설정 저장
ipcMain.handle('save-config', async (event, config) => {
    try {
        await fs.promises.writeFile(configPath, JSON.stringify(config, null, 2), 'utf8');
        return { success: true };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
});

// 최근 프로젝트 로드
ipcMain.handle('load-recent-projects', async () => {
    try {
        if (fs.existsSync(recentProjectsPath)) {
            const data = await fs.promises.readFile(recentProjectsPath, 'utf8');
            return {
                success: true,
                projects: JSON.parse(data)
            };
        }
        
        return {
            success: true,
            projects: []
        };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
});

// 최근 프로젝트 저장
ipcMain.handle('save-recent-project', async (event, project) => {
    try {
        let recentProjects = [];
        
        // 기존 최근 프로젝트들 로드
        if (fs.existsSync(recentProjectsPath)) {
            const data = await fs.promises.readFile(recentProjectsPath, 'utf8');
            recentProjects = JSON.parse(data);
        }
        
        // 중복 제거 및 새 프로젝트 추가
        recentProjects = recentProjects.filter(p => p.path !== project.path);
        recentProjects.unshift({
            ...project,
            lastOpened: new Date().toISOString()
        });
        
        // 최대 10개까지만 유지
        recentProjects = recentProjects.slice(0, 10);
        
        await fs.promises.writeFile(recentProjectsPath, JSON.stringify(recentProjects, null, 2), 'utf8');
        
        return { success: true };
    } catch (error) {
        return {
            success: false,
            error: error.message
        };
    }
});

// ====== 알림 및 시스템 ======

// 시스템 알림 표시
ipcMain.handle('show-notification', async (event, title, body) => {
    const { Notification } = require('electron');
    
    // ⭐⭐ 추가: 알림 설정 확인
    try {
        let notificationsEnabled = true; // 기본값
        
        // 설정 파일에서 알림 설정 로드
        if (fs.existsSync(configPath)) {
            const configData = await fs.promises.readFile(configPath, 'utf8');
            const config = JSON.parse(configData);
            notificationsEnabled = config.notifications !== undefined ? config.notifications : true;
        }
        
        // 알림이 비활성화되어 있으면 표시하지 않음
        if (!notificationsEnabled) {
            console.log('[Notification] 알림이 비활성화되어 있어 표시하지 않습니다.');
            return { success: false, reason: 'notifications_disabled' };
        }
        
        // 알림 표시
        if (Notification.isSupported()) {
            new Notification({
                title: title,
                body: body,
                icon: path.join(__dirname, 'assets', 'icon.png')
            }).show();
            
            return { success: true };
        }
        
        return { success: false, reason: 'not_supported' };
        
    } catch (error) {
        console.error('[Notification] 오류:', error);
        return { success: false, error: error.message };
    }
});

// 앱 정보 가져오기
ipcMain.handle('get-app-info', () => {
    return {
        name: app.getName(),
        version: app.getVersion(),
        electronVersion: process.versions.electron,
        nodeVersion: process.versions.node,
        platform: process.platform
    };
});

// Finder에서 보기
ipcMain.handle('show-in-finder', async (event, folderPath) => {
    try {
        const { shell } = require('electron');
        await shell.showItemInFolder(folderPath);
        return { success: true };
    } catch (error) {
        console.error('Finder에서 보기 실패:', error);
        return { 
            success: false, 
            error: error.message 
        };
    }
});

// 개발자 도구 토글
ipcMain.on('toggle-dev-tools', () => {
    if (mainWindow) {
        mainWindow.webContents.toggleDevTools();
    }
});

// 앱 재시작
ipcMain.on('restart-app', () => {
    app.relaunch();
    app.exit();
});

// 에러 핸들링
process.on('uncaughtException', (error) => {
    console.error('Uncaught Exception:', error);
});

process.on('unhandledRejection', (error) => {
    console.error('Unhandled Rejection:', error);
});
