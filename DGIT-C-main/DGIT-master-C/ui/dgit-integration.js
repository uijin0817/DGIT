const { spawn, exec } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

/**
 * DGit CLI 연동 클래스
 * DGit 명령어들을 래핑하고 Electron 앱에서 사용할 수 있도록 함
 */
class DGitIntegration {
    constructor() {
        this.dgitPath = this.findDGitPath();
        this.isAvailable = false;
        this.version = null;
        this.init();
    }

    /**
     * DGit CLI 초기화 및 사용 가능 여부 확인
     */
    async init() {
        try {
            console.log(`[DGit] Initializing with path: ${this.dgitPath}`);

            // DGit 실행 파일 존재 여부 확인
            if (this.dgitPath !== 'dgit' && !fs.existsSync(this.dgitPath)) {
                console.warn(`[DGit] Executable not found at: ${this.dgitPath}`);
                this.isAvailable = false;
                return;
            }

            // DGit은 --version 플래그를 지원하지 않으므로 help 명령으로 테스트
            const result = await this.executeCommand('help');
            this.isAvailable = true;
            this.version = 'DGit CLI Available';
            console.log(`[DGit] CLI found and available at ${this.dgitPath}`);
        } catch (error) {
            console.warn('[DGit] CLI not available:', error.message);
            this.isAvailable = false;
        }
    }

    /**
     * DGit CLI 경로 찾기
     * 여러 가능한 위치에서 DGit 실행파일을 찾음
     */
    findDGitPath() {
        const platform = process.platform;
        const isWindows = platform === 'win32';
        const isMac = platform === 'darwin';
        const isLinux = platform === 'linux';

        console.log(`[DGit] Searching for DGit CLI on ${platform}`);

        const possiblePaths = [];

        // Windows 전용 경로들
        if (isWindows) {
            // 프로젝트 구조에서 Windows 바이너리 찾기
            possiblePaths.push(
                // 상위 디렉토리의 dgit 폴더 (프로젝트 루트)
                path.join(__dirname, '..', '..', 'dgit', 'dgit.exe'),
                path.join(__dirname, '..', '..', 'dgit', 'dgit-windows.exe'),
                path.join(__dirname, '..', '..', 'dgit', 'dgit-win.exe'),
                path.join(__dirname, '..', '..', 'dgit.exe'),

                // ui 폴더 기준
                path.join(__dirname, '..', 'dgit', 'dgit.exe'),
                path.join(__dirname, '..', 'dgit', 'dgit-windows.exe'),
                path.join(__dirname, '..', 'dgit.exe'),

                // 현재 디렉토리
                path.join(__dirname, 'dgit.exe'),
                path.join(process.cwd(), 'dgit.exe'),

                // 사용자 데스크탑
                path.join(os.homedir(), 'Desktop', 'DGIT', 'dgit', 'dgit.exe'),
                path.join(os.homedir(), 'Desktop', 'DGIT', 'dgit.exe'),
                path.join(os.homedir(), 'Desktop', 'dgit.exe'),

                // 일반적인 Windows 설치 경로
                'C:\\Program Files\\DGit\\dgit.exe',
                'C:\\Program Files (x86)\\DGit\\dgit.exe',
                path.join(os.homedir(), 'AppData', 'Local', 'DGit', 'dgit.exe'),

                // 개발 환경 경로
                'C:\\dgit\\dgit.exe',
                'D:\\dgit\\dgit.exe'
            );
        }

        // macOS 전용 경로들
        if (isMac) {
            possiblePaths.push(
                // 프로젝트 구조에서 Mac 바이너리 찾기
                path.join(__dirname, '..', '..', 'dgit', 'dgit-mac'),
                path.join(__dirname, '..', '..', 'dgit', 'dgit'),
                path.join(__dirname, '..', 'dgit', 'dgit-mac'),
                path.join(__dirname, '..', 'dgit', 'dgit'),

                // 사용자 데스크탑
                path.join(os.homedir(), 'Desktop', 'DGIT-MAC-master', 'dgit', 'dgit'),
                path.join(os.homedir(), 'Desktop', 'DGIT', 'dgit', 'dgit'),
                path.join(os.homedir(), 'Desktop', 'DGIT', 'dgit'),

                // 일반적인 macOS 설치 경로
                '/usr/local/bin/dgit',
                '/opt/homebrew/bin/dgit',
                path.join(os.homedir(), '.local', 'bin', 'dgit'),
                path.join(os.homedir(), 'bin', 'dgit')
            );
        }

        // Linux 전용 경로들
        if (isLinux) {
            possiblePaths.push(
                path.join(__dirname, '..', '..', 'dgit', 'dgit-linux'),
                path.join(__dirname, '..', '..', 'dgit', 'dgit'),
                path.join(__dirname, '..', 'dgit', 'dgit-linux'),
                path.join(__dirname, '..', 'dgit', 'dgit'),
                '/usr/local/bin/dgit',
                '/usr/bin/dgit',
                path.join(os.homedir(), '.local', 'bin', 'dgit'),
                path.join(os.homedir(), 'bin', 'dgit')
            );
        }

        // 번들된 바이너리 (앱과 함께 패키징된 경우)
        possiblePaths.push(
            path.join(process.resourcesPath, 'bin', 'dgit' + (isWindows ? '.exe' : '')),
            path.join(__dirname, 'bin', 'dgit' + (isWindows ? '.exe' : ''))
        );

        // 실제로 존재하는 경로 찾기
        for (const dgitPath of possiblePaths) {
            try {
                console.log(`[DGit] Checking path: ${dgitPath}`);

                if (fs.existsSync(dgitPath)) {
                    console.log(`[DGit] Found executable at: ${dgitPath}`);

                    // Windows에서는 실행 권한 체크 불필요
                    if (!isWindows) {
                        const stats = fs.statSync(dgitPath);
                        if (!(stats.mode & parseInt('111', 8))) {
                            console.log(`[DGit] No execute permission for: ${dgitPath}`);
                            continue; // 실행 권한이 없음
                        }
                    }

                    return dgitPath;
                }
            } catch (error) {
                console.log(`[DGit] Error checking path ${dgitPath}:`, error.message);
                continue; // 다음 경로 시도
            }
        }

        // PATH 환경 변수에서 찾기 (마지막 수단)
        console.log('[DGit] No dgit executable found in predefined paths, trying PATH environment');

        // Windows의 경우 dgit.exe도 시도
        if (isWindows) {
            // PATH에서 dgit.exe 찾기 시도
            try {
                const { execSync } = require('child_process');
                const result = execSync('where dgit.exe', { encoding: 'utf8' });
                const dgitPath = result.split('\n')[0].trim();
                if (dgitPath && fs.existsSync(dgitPath)) {
                    console.log(`[DGit] Found in PATH: ${dgitPath}`);
                    return dgitPath;
                }
            } catch (error) {
                console.log('[DGit] dgit.exe not found in PATH');
            }

            // dgit (확장자 없이) 시도
            try {
                const { execSync } = require('child_process');
                const result = execSync('where dgit', { encoding: 'utf8' });
                const dgitPath = result.split('\n')[0].trim();
                if (dgitPath && fs.existsSync(dgitPath)) {
                    console.log(`[DGit] Found in PATH: ${dgitPath}`);
                    return dgitPath;
                }
            } catch (error) {
                console.log('[DGit] dgit not found in PATH');
            }
        }

        // PATH에서 찾기 (기본값)
        console.log('[DGit] Falling back to default "dgit" command');
        return 'dgit';
    }

    /**
     * DGit CLI가 사용 가능한지 확인
     */
    isReady() {
        return this.isAvailable;
    }

    /**
     * DGit 버전 정보 반환
     */
    getVersion() {
        return this.version;
    }

    /**
     * DGit 명령어 실행
     * @param {string} command - 실행할 명령어
     * @param {Array} args - 명령어 인수들
     * @param {string} cwd - 작업 디렉토리
     * @param {Object} options - 추가 옵션들
     */
    async executeCommand(command, args = [], cwd = null, options = {}) {
        return new Promise((resolve, reject) => {
            const allArgs = command ? [command, ...args] : args;
            const spawnOptions = {
                cwd: cwd || process.cwd(),
                env: { ...process.env, ...options.env },
                shell : false, // ⭐ 한글 경로 지원을 위해 shell 비활성화
                // shell: process.platform === 'win32', // Windows에서 shell 옵션 활성화
                windowsHide: true, // Windows에서 콘솔 창 숨기기
                ...options
            };

            console.log(`[DGit] Executing: ${this.dgitPath} ${allArgs.join(' ')}`);
            console.log(`[DGit] Working directory: ${spawnOptions.cwd}`);

            const childProcess = spawn(this.dgitPath, allArgs, spawnOptions);

            let stdout = '';
            let stderr = '';

            // 출력 스트림 처리
            if (childProcess.stdout) {
                childProcess.stdout.on('data', (data) => {
                    const output = data.toString();
                    stdout += output;
                    if (options.onOutput) {
                        options.onOutput(output, 'stdout');
                    }
                });
            }

            if (childProcess.stderr) {
                childProcess.stderr.on('data', (data) => {
                    const output = data.toString();
                    stderr += output;
                    if (options.onOutput) {
                        options.onOutput(output, 'stderr');
                    }
                });
            }

            // 프로세스 종료 처리
            childProcess.on('close', (code, signal) => {
                console.log(`[DGit] Command finished with code: ${code}, signal: ${signal}`);

                if (code === 0) {
                    resolve({
                        output: stdout,
                        error: stderr,
                        code: code,
                        success: true
                    });
                } else {
                    reject({
                        output: stdout,
                        error: stderr || `Process exited with code ${code}`,
                        code: code,
                        success: false
                    });
                }
            });

            // 에러 처리
            childProcess.on('error', (error) => {
                console.error('[DGit] Process error:', error);

                // ENOENT 오류에 대한 더 자세한 메시지
                if (error.code === 'ENOENT') {
                    const errorMessage = `DGit executable not found at: ${this.dgitPath}. Please ensure DGit is installed and accessible.`;
                    console.error('[DGit]', errorMessage);
                    reject({
                        output: stdout,
                        error: errorMessage,
                        code: -1,
                        success: false
                    });
                } else {
                    reject({
                        output: stdout,
                        error: error.message,
                        code: -1,
                        success: false
                    });
                }
            });

            // 타임아웃 처리
            if (options.timeout) {
                setTimeout(() => {
                    childProcess.kill('SIGTERM');
                    reject({
                        output: stdout,
                        error: 'Command timed out',
                        code: -1,
                        success: false
                    });
                }, options.timeout);
            }
        });
    }

    // ====== DGit 기본 명령어들 ======

    /**
     * 저장소 초기화
     */
    async init(projectPath) {
        return this.executeCommand('init', [], projectPath);
    }

    /**
     * 파일 상태 확인
     */
    async status(projectPath) {
        return this.executeCommand('status', [], projectPath);
    }

    /**
     * 파일 추가 (스테이징)
     */
    async add(projectPath, files = ['.']) {
        const fileArgs = Array.isArray(files) ? files : [files];
        return this.executeCommand('add', fileArgs, projectPath);
    }

    /**
     * 변경사항 커밋
     */
       /**
     * 변경사항 커밋
     */
       async commit(projectPath, message, options = {}) {
        // ⭐⭐ 수정: 메시지를 따옴표로 감싸서 띄어쓰기 포함
        const args = ['-m', `"${message}"`];

        // 추가 옵션들
        if (options.author) {
            args.push('--author', options.author);
        }
        if (options.amend) {
            args.push('--amend');
        }
        if (options.noEdit) {
            args.push('--no-edit');
        }

        return this.executeCommand('commit', args, projectPath);
    }

    /**
     * 커밋 히스토리 조회
     */
    async log(projectPath, options = {}) {
        const args = [];

        // 옵션 처리
        if (options.limit) {
            args.push(`--max-count=${options.limit}`);
        }
        if (options.oneline) {
            args.push('--oneline');
        }
        if (options.format) {
            args.push(`--pretty=${options.format}`);
        }
        if (options.since) {
            args.push(`--since="${options.since}"`);
        }
        if (options.until) {
            args.push(`--until="${options.until}"`);
        }

        return this.executeCommand('log', args, projectPath);
    }

    /**
     * 파일/커밋 차이점 보기
     */
    async diff(projectPath, options = {}) {
        const args = [];

        if (options.staged) {
            args.push('--staged');
        }
        if (options.cached) {
            args.push('--cached');
        }
        if (options.nameOnly) {
            args.push('--name-only');
        }
        if (options.statOnly) {
            args.push('--stat');
        }
        if (options.commit1 && options.commit2) {
            args.push(options.commit1, options.commit2);
        } else if (options.commit1) {
            args.push(options.commit1);
        }
        if (options.files) {
            args.push('--', ...options.files);
        }

        return this.executeCommand('diff', args, projectPath);
    }

    /**
     * 파일 복원
     */
    async restore(projectPath, files, options = {}) {
        const args = [];

        if (options.staged) {
            args.push('--staged');
        }
        if (options.source) {
            args.push('--source', options.source);
        }
        if (options.worktree) {
            args.push('--worktree');
        }

        // 파일들 추가
        const fileList = Array.isArray(files) ? files : [files];
        args.push(...fileList);

        return this.executeCommand('restore', args, projectPath);
    }

    /**
     * 특정 커밋의 파일들 복원
     */
    async reset(projectPath, target = 'HEAD', options = {}) {
        const args = [];

        if (options.hard) {
            args.push('--hard');
        } else if (options.soft) {
            args.push('--soft');
        } else if (options.mixed) {
            args.push('--mixed');
        }

        args.push(target);

        if (options.files) {
            args.push('--', ...options.files);
        }

        return this.executeCommand('reset', args, projectPath);
    }

    // ====== 유틸리티 함수들 ======

    /**
     * 저장소인지 확인
     */
    async isRepository(projectPath) {
        try {
            // DGit CLI가 사용 가능한지 먼저 확인
            if (!this.isAvailable) {
                console.warn('[DGit] CLI is not available');
                return false;
            }

            const result = await this.executeCommand('status', [], projectPath);
            return result.success;
        } catch (error) {
            console.log(`[DGit] Repository check failed for ${projectPath}:`, error.message);
            return false;
        }
    }

    /**
     * 변경된 파일 목록 가져오기
     */
    async getChangedFiles(projectPath) {
        try {
            const result = await this.status(projectPath);
            // 상태 출력을 파싱하여 변경된 파일 목록 반환
            return this.parseStatusOutput(result.output);
        } catch (error) {
            return [];
        }
    }

    /**
     * 상태 출력 파싱
     */
    parseStatusOutput(output) {
        const files = [];
        const lines = output.split('\n');

        for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed) continue;

            // DGit 상태 출력 형식에 맞게 파싱
            // 예: "M  filename.psd" 또는 "A  newfile.ai"
            const match = trimmed.match(/^([MAD?!])\s+(.+)$/);
            if (match) {
                const [, status, filename] = match;
                files.push({
                    filename: filename,
                    status: this.getStatusText(status),
                    statusCode: status
                });
            }
        }

        return files;
    }

    /**
     * 상태 코드를 텍스트로 변환
     */
    getStatusText(statusCode) {
        const statusMap = {
            'M': 'modified',
            'A': 'added',
            'D': 'deleted',
            'R': 'renamed',
            'C': 'copied',
            'U': 'updated',
            '?': 'untracked',
            '!': 'ignored'
        };
        return statusMap[statusCode] || 'unknown';
    }

    /**
     * 커밋 로그를 파싱하여 구조화된 데이터 반환
     */
    parseLogOutput(output) {
        const commits = [];
        const commitBlocks = output.split('\n\n').filter(block => block.trim());

        for (const block of commitBlocks) {
            const lines = block.split('\n');
            const commit = {};

            for (const line of lines) {
                if (line.startsWith('commit ')) {
                    commit.hash = line.substring(7).trim();
                } else if (line.startsWith('Author: ')) {
                    commit.author = line.substring(8).trim();
                } else if (line.startsWith('Date: ')) {
                    commit.date = line.substring(6).trim();
                } else if (line.trim() && !line.startsWith(' ')) {
                    // 커밋 메시지
                    commit.message = (commit.message || '') + line.trim() + ' ';
                }
            }

            if (commit.hash) {
                commit.message = (commit.message || '').trim();
                commits.push(commit);
            }
        }

        return commits;
    }

    /**
     * DGit 실행 파일의 전체 경로 반환
     */
    getExecutablePath() {
        return this.dgitPath;
    }

    /**
     * DGit CLI 재검색 및 재초기화
     */
    async reinitialize() {
        console.log('[DGit] Reinitializing DGit CLI...');
        this.dgitPath = this.findDGitPath();
        await this.init();
        return this.isAvailable;
    }
}

module.exports = DGitIntegration;