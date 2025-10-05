// 프로젝트 관련 함수들

// DGit CLI 사용 가능 여부 확인
async function checkDGitAvailability() {
    try {
        const result = await window.electron.dgit.command('help');
        return result.success;
    } catch (error) {
        return false;
    }
}

// 저장소 확인
async function checkIfRepository(projectPath) {
    try {
        const result = await window.electron.dgit.status(projectPath);
        return result.success;
    } catch (error) {
        console.log('Repository check error:', error);
        return false;
    }
}

// DGit 없이 프로젝트 선택
async function selectProjectWithoutDGit() {
    closeModal();

    try {
        const result = await window.electron.selectFolder();

        if (result.success) {
            const projectInfo = {
                name: result.name,
                path: result.path
            };

            await openProjectWithoutDGit(projectInfo);
        } else {
            showToast('폴더 선택이 취소되었습니다', 'warning');
        }
    } catch (error) {
        console.error('프로젝트 선택 실패:', error);
        showToast('프로젝트 선택 중 오류가 발생했습니다', 'error');
    }
}

// DGit 기능 없이 프로젝트 열기
async function openProjectWithoutDGit(projectInfo) {
    closeModal();

    currentProject = projectInfo;
    document.getElementById('homeScreen').style.display = 'none';
    document.getElementById('workspace').classList.add('active');
    document.getElementById('projectName').textContent = projectInfo.name;
    document.getElementById('projectPath').textContent = projectInfo.path;

    // 최근 프로젝트에 저장
    try {
        await window.electron.recentProjects.save(projectInfo);
        await loadRecentProjects();
    } catch (error) {
        console.error('최근 프로젝트 저장 실패:', error);
    }

    // init 버튼 보이기 (DGit 저장소가 없으므로)
    const initButton = document.getElementById('initButton');
    if (initButton) {
        initButton.style.display = 'inline-block';
    }

    // 파일만 로드 (DGit 기능 제외)
    await loadProjectFilesOnly();

    showToast(`프로젝트 '${projectInfo.name}'을 열었습니다 (DGit 기능 없음)`, 'warning');
}

// 파일만 로드 (DGit 기능 제외)
async function loadProjectFilesOnly() {
    if (!currentProject) return;

    try {
        const result = await window.electron.scanDirectory(currentProject.path);

        if (result.success) {
            const files = result.files.map(file => ({
                name: file.name,
                type: file.type,
                size: formatFileSize(file.size),
                modified: formatDate(file.modified),
                status: 'unknown',
                path: file.path
            }));

            renderFiles(files);

            // 빈 커밋 히스토리와 상태 표시
            renderCommits([]);
            updateTerminalStatus('DGit CLI를 사용할 수 없습니다. 파일 보기만 가능합니다.');
        } else {
            showToast('파일을 스캔할 수 없습니다', 'error');
        }
    } catch (error) {
        console.error('파일 로드 실패:', error);
        showToast('파일 로드 중 오류가 발생했습니다', 'error');
    }
}

// 저장소 초기화
async function initializeRepository(projectInfo) {
    closeModal();

    try {
        showToast('DGit 저장소를 초기화하는 중...', 'info');

        const result = await window.electron.dgit.init(projectInfo.path);

        if (result.success) {
            showToast('DGit 저장소가 초기화되었습니다', 'success');
            await openProjectDirectly(projectInfo);
        } else {
            showToast(`저장소 초기화에 실패했습니다: ${result.error}`, 'error');
        }
    } catch (error) {
        console.error('저장소 초기화 실패:', error);
        showToast('저장소 초기화 중 오류가 발생했습니다', 'error');
    }
}

// 현재 프로젝트 초기화 (GUI 버튼용)
async function initializeCurrentProject() {
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    // 확인 모달 표시
    showModal('DGit 저장소 초기화', '현재 프로젝트를 DGit 저장소로 초기화하시겠습니까?', `
        <div style="padding: 20px; text-align: center;">
            <div style="margin-bottom: 20px;">
                <div style="font-size: 3rem; margin-bottom: 16px;">⚡</div>
                <h3 style="margin-bottom: 12px; color: var(--text-primary);">DGit 저장소 초기화</h3>
                <p style="color: var(--text-secondary); line-height: 1.5;">
                    현재 프로젝트에 DGit 저장소를 생성합니다.<br>
                    이 작업은 프로젝트 폴더에 .dgit 폴더를 생성하여<br>
                    버전 관리를 시작할 수 있게 합니다.
                </p>
            </div>
            <div style="display: flex; gap: 12px; justify-content: center;">
                <button class="btn btn-secondary" onclick="closeModal()">취소</button>
                <button class="btn btn-warning" onclick="executeInitialization()">
                    초기화 실행
                </button>
            </div>
        </div>
    `);
}

// 초기화 실행
async function executeInitialization() {
    closeModal();

    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    try {
        showToast('DGit 저장소를 초기화하는 중...', 'info');

        const result = await window.electron.dgit.init(currentProject.path);

        if (result.success) {
            showToast('DGit 저장소가 초기화되었습니다', 'success');
            
            // init 버튼 숨기기
            const initButton = document.getElementById('initButton');
            if (initButton) {
                initButton.style.display = 'none';
            }
            
            // 프로젝트 데이터 다시 로드
            await loadProjectData();
        } else {
            showToast(`저장소 초기화에 실패했습니다: ${result.error}`, 'error');
        }
    } catch (error) {
        console.error('저장소 초기화 실패:', error);
        showToast('저장소 초기화 중 오류가 발생했습니다', 'error');
    }
}

// 프로젝트 열기
async function openProject(projectInfo) {
    // DGit 저장소 존재 여부 확인
    try {
        const statusResult = await window.electron.dgit.command('status', [], projectInfo.path);

        // DGit 저장소가 없는 경우 초기화 프롬프트 표시
        if (!statusResult.success && statusResult.error && statusResult.error.includes('not a DGit repository')) {
            showDGitInitPrompt(projectInfo);
            return;
        }
    } catch (error) {
        // DGit 명령 실행 실패 시 초기화 프롬프트 표시
        showDGitInitPrompt(projectInfo);
        return;
    }

    // DGit 저장소가 있는 경우 정상적으로 프로젝트 열기
    await openProjectDirectly(projectInfo);
}

// DGit 초기화 프롬프트 표시
function showDGitInitPrompt(projectInfo) {
    showModal('DGit 저장소 초기화', '이 프로젝트에는 DGit 저장소가 없습니다', `
        <div style="padding: 20px; text-align: center;">
            <div style="margin-bottom: 20px;">
                <div style="font-size: 3rem; margin-bottom: 16px;">📦</div>
                <h3 style="margin-bottom: 12px; color: var(--text-primary);">DGit 저장소가 필요합니다</h3>
                <p style="color: var(--text-secondary); line-height: 1.5;">
                    이 프로젝트에서 버전 관리를 사용하려면<br>
                    DGit 저장소를 초기화해야 합니다.
                </p>
            </div>
            <div style="display: flex; gap: 12px; justify-content: center;">
                <button class="btn btn-secondary" onclick="closeModal()">취소</button>
                <button class="btn btn-primary" onclick="initializeRepository(${JSON.stringify(projectInfo).replace(/"/g, '&quot;')})">
                    저장소 초기화
                </button>
                <button class="btn btn-secondary" onclick="openProjectWithoutDGit(${JSON.stringify(projectInfo).replace(/"/g, '&quot;')})">
                    DGit 없이 열기
                </button>
            </div>
        </div>
    `);
}

// 프로젝트 직접 열기 (DGit 저장소 확인 완료 후)
async function openProjectDirectly(projectInfo) {
    currentProject = projectInfo;
    document.getElementById('homeScreen').style.display = 'none';
    document.getElementById('workspace').classList.add('active');
    document.getElementById('projectName').textContent = projectInfo.name;
    document.getElementById('projectPath').textContent = projectInfo.path;

    // 최근 프로젝트에 저장
    try {
        await window.electron.recentProjects.save(projectInfo);
        await loadRecentProjects(); // 목록 새로고침
    } catch (error) {
        console.error('최근 프로젝트 저장 실패:', error);
    }

    // DGit 저장소 상태 확인 및 init 버튼 표시 여부 결정
    await checkRepositoryStatusAndShowInitButton();

    // 프로젝트 데이터 로드
    await loadProjectData();

    showToast(`프로젝트 '${projectInfo.name}'을 열었습니다`, 'success');
}

// DGit 저장소 상태 확인 및 init 버튼 표시
async function checkRepositoryStatusAndShowInitButton() {
    if (!currentProject) return;

    const initButton = document.getElementById('initButton');
    if (!initButton) return;

    try {
        const isRepo = await checkIfRepository(currentProject.path);
        
        if (isRepo) {
            // DGit 저장소가 있으면 init 버튼 숨기기
            initButton.style.display = 'none';
        } else {
            // DGit 저장소가 없으면 init 버튼 보이기
            initButton.style.display = 'inline-block';
        }
    } catch (error) {
        console.error('저장소 상태 확인 실패:', error);
        // 오류 시 init 버튼 보이기 (안전한 기본값)
        initButton.style.display = 'inline-block';
    }
}

// 프로젝트 데이터 로드
async function loadProjectData() {
    if (!currentProject) return;

    try {
        // DGit 저장소 상태 재확인 및 init 버튼 업데이트
        await checkRepositoryStatusAndShowInitButton();

        // 프로젝트 파일 스캔
        await loadProjectFiles();

        // 커밋 히스토리 로드
        await loadCommitHistory();

        // DGit 상태 확인
        await updateProjectStatus();

    } catch (error) {
        console.error('프로젝트 데이터 로드 실패:', error);
        showToast('프로젝트 데이터를 로드하는 중 오류가 발생했습니다', 'error');
    }
}

// 프로젝트 파일 로드 (프로그레스 바 포함)
async function loadProjectFiles() {
    if (!currentProject) return;

    try {
        // 초기 프로그레스 바 표시
        showProgressBar('fileList', 0, '프로젝트 스캔 시작...');
        
        // ⭐⭐ 주석 처리: 터미널 히스토리 유지를 위해 메시지 추가 안 함
        // const terminalStatus = document.getElementById('terminalStatus');
        // terminalStatus.innerHTML += `
        //     <div style="margin-bottom: 8px;">
        //         <span style="color: var(--accent-blue);">📂</span>
        //         <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
        //         파일 스캔 시작...
        //     </div>
        // `;
        // terminalStatus.scrollTop = terminalStatus.scrollHeight;

        // 1단계: 디렉토리 스캔 시작 (0% ~ 20%)
        showProgressBar('fileList', 5, '디렉토리를 분석하는 중...');

        const result = await window.electron.scanDirectory(currentProject.path);

        if (result.success) {
            const totalFiles = result.files.length;

            // 파일이 없는 경우 예외 처리
            if (totalFiles === 0) {
                hideLoadingSpinner('fileList');
                renderFiles([]);
                return;
            }

            // 2단계: 파일 목록 로드 완료 (20%)
            showProgressBar('fileList', 20, `${totalFiles}개 파일 발견`);

            // 3단계: 파일 정보 처리 (20% ~ 70%)
            let processedFiles = 0;
            const files = [];

            for (const file of result.files) {
                // 각 파일 처리
                const processedFile = {
                    name: file.name,
                    type: file.type,
                    size: formatFileSize(file.size),
                    modified: formatDate(file.modified),
                    status: 'unknown',
                    path: file.path
                };

                files.push(processedFile);
                processedFiles++;

                // 진행률 계산 (20% ~ 70% 범위)
                const fileProgress = 20 + Math.round((processedFiles / totalFiles) * 50);
                showProgressBar('fileList', fileProgress, `파일 정보 처리 중... ${processedFiles}/${totalFiles}`);

                // UI 블로킹 방지를 위한 비동기 처리
                if (processedFiles % 10 === 0) {
                    await new Promise(resolve => setTimeout(resolve, 1));
                }
            }

            // 4단계: Git 상태 확인 (70% ~ 90%)
            showProgressBar('fileList', 75, 'Git 상태 확인 중...');

            // DGit 상태로 파일 상태 업데이트
            await updateFileStatuses(files);

            // 5단계: 렌더링 준비 (90% ~ 100%)
            showProgressBar('fileList', 95, '파일 목록 렌더링 중...');

            // 잠시 대기 후 완료
            await new Promise(resolve => setTimeout(resolve, 200));
            showProgressBar('fileList', 100, `${totalFiles}개 파일 로드 완료!`);

            // 상태 터미널에 파일 스캔 완료 메시지 추가
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 8px;">
                    <span style="color: var(--accent-green);">✅</span>
                    <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
                    파일 스캔 완료: ${totalFiles}개 파일 발견
                </div>
            `;
            terminalStatus.scrollTop = terminalStatus.scrollHeight;

            // 잠시 후 실제 파일 목록 표시
            setTimeout(() => {
                hideLoadingSpinner('fileList');
                renderFiles(files);
            }, 800);

        } else {
            // 상태 터미널에 스캔 실패 메시지 추가
            const terminalStatus = document.getElementById('terminalStatus');
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 8px;">
                    <span style="color: var(--accent-red);">❌</span>
                    <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
                    파일 스캔 실패: 디렉토리를 읽을 수 없습니다
                </div>
            `;
            terminalStatus.scrollTop = terminalStatus.scrollHeight;
            
            hideLoadingSpinner('fileList');
            showToast('파일을 스캔할 수 없습니다', 'error');
        }
    } catch (error) {
        console.error('파일 로드 실패:', error);
        
        // 상태 터미널에 오류 메시지 추가
        const terminalStatus = document.getElementById('terminalStatus');
        terminalStatus.innerHTML += `
            <div style="margin-bottom: 8px;">
                <span style="color: var(--accent-red);">❌</span>
                <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
                파일 로드 오류: ${error.message}
            </div>
        `;
        terminalStatus.scrollTop = terminalStatus.scrollHeight;
        
        hideLoadingSpinner('fileList');
        showToast('파일 로드 중 오류가 발생했습니다', 'error');
    }
}

// 커밋 히스토리 로드
async function loadCommitHistory() {
    if (!currentProject) return;

    try {
        const result = await window.electron.dgit.log(currentProject.path, 10);

        if (result.success) {
            const commits = parseCommitLog(result.output);
            renderCommits(commits);
        } else {
            // 저장소가 초기화되지 않은 경우
            renderCommits([]);
        }
    } catch (error) {
        console.error('커밋 히스토리 로드 실패:', error);
        renderCommits([]);
    }
}

// 프로젝트 상태 업데이트
async function updateProjectStatus() {
    if (!currentProject) return;

    try {
        const result = await window.electron.dgit.status(currentProject.path);

        if (result.success) {
            // ⭐⭐ 수정: 상태 메시지 표시 안 함 (주석 처리)
            // updateTerminalStatus(result.output);
        } else {
            // updateTerminalStatus('DGit 저장소가 초기화되지 않았습니다.');
        }
    } catch (error) {
        console.error('상태 업데이트 실패:', error);
        // updateTerminalStatus('상태를 확인할 수 없습니다.');
    }
}

// 파일 상태 업데이트
async function updateFileStatuses(files) {
    if (!currentProject) return;

    try {
        // 먼저 저장소인지 확인
        const isRepo = await checkIfRepository(currentProject.path);
        
        if (!isRepo) {
            // DGit 저장소가 아니면 모든 파일을 'untracked'로 설정
            files.forEach(file => {
                file.status = 'untracked';
            });
            return;
        }

        const result = await window.electron.dgit.status(currentProject.path);
        if (result.success) {
            const statusMap = parseGitStatus(result.output);

            files.forEach(file => {
                if (statusMap[file.name]) {
                    file.status = statusMap[file.name];
                } else {
                    // 상태 맵에 없으면 커밋된 파일로 간주
                    file.status = 'committed';
                }
            });
        } else {
            // status 명령 실패 시 모든 파일을 'untracked'로 설정
            files.forEach(file => {
                file.status = 'untracked';
            });
        }
    } catch (error) {
        console.error('파일 상태 업데이트 실패:', error);
        // 오류 시 모든 파일을 'unknown'으로 설정
        files.forEach(file => {
            file.status = 'unknown';
        });
    }
}

// 프로젝트 변경
async function changeProject() {
    await selectNewProject();
}

// ⭐⭐ 수정: 프로젝트 스캔 - CLI 스타일 출력 + 히스토리 유지
async function scanProject() {
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    try {
        // 상태 탭으로 자동 전환
        showTerminalTab('status');
        
        const terminalStatus = document.getElementById('terminalStatus');
        const scanStartTime = Date.now();
        
        // ⭐ 히스토리 유지: innerHTML = 대신 += 사용
        // 스캔 시작 메시지 추가
        terminalStatus.innerHTML += `
            <div style="margin-bottom: 16px; padding: 12px; background: var(--bg-tertiary); border-radius: 6px; border-left: 3px solid var(--accent-blue);">
                <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
                    <span style="color: var(--accent-blue); font-size: 1.2rem;">🔍</span>
                    <span style="font-weight: bold; color: var(--text-primary);">Scanning design files in: ${currentProject.path}</span>
                </div>
                <div style="color: var(--text-secondary); font-size: 0.85rem;">
                    <span>[${new Date().toLocaleTimeString()}]</span>
                    <span style="margin-left: 8px;">프로젝트: ${currentProject.name}</span>
                </div>
            </div>
        `;
        terminalStatus.scrollTop = terminalStatus.scrollHeight;
        
        showToast('프로젝트를 스캔하고 있습니다...', 'info');
        
        // 파일 스캔 실행
        const scanResult = await window.electron.scanDirectory(currentProject.path);
        
        if (scanResult.success && scanResult.files) {
            const files = scanResult.files;
            const totalFiles = files.length;
            const scanEndTime = Date.now();
            const scanDuration = scanEndTime - scanStartTime;
            
            // 총 용량 계산
            const totalSize = files.reduce((sum, f) => sum + (f.size || 0), 0);
            const totalSizeMB = (totalSize / (1024 * 1024)).toFixed(1);
            
            // 파일 타입별 분류
            const filesByType = {};
            files.forEach(file => {
                const ext = file.type || 'unknown';
                if (!filesByType[ext]) {
                    filesByType[ext] = [];
                }
                filesByType[ext].push(file);
            });
            
            // CLI 스타일 출력
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 16px; padding: 12px; background: var(--bg-secondary); border-left: 3px solid var(--accent-green); border-radius: 4px;">
                    <div style="margin-bottom: 12px;">
                        <span style="color: var(--accent-green); font-size: 1.1rem;">✓</span>
                        <span style="font-weight: bold; color: var(--text-primary); margin-left: 8px;">
                            Found ${totalFiles} design files (${totalSizeMB} MB)
                        </span>
                    </div>
                    
                    ${Object.entries(filesByType).length > 0 ? `
                        <div style="margin-left: 20px; margin-bottom: 12px;">
                            ${Object.entries(filesByType).map(([type, typeFiles]) => `
                                <div style="margin-bottom: 4px; color: var(--text-secondary); font-size: 0.9rem;">
                                    <span style="color: var(--accent-blue);">•</span>
                                    <span style="margin-left: 8px;">${typeFiles.length} ${type.toUpperCase()} files</span>
                                </div>
                            `).join('')}
                        </div>
                    ` : ''}
                    
                    <div style="margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--border-color); color: var(--text-secondary); font-size: 0.85rem;">
                        <div>Scan completed in ${scanDuration}ms</div>
                        <div style="margin-top: 4px; color: var(--text-tertiary);">
                            Use 'dgit show &lt;filename&gt;' for detailed file analysis
                        </div>
                    </div>
                </div>
            `;
            
            // ⭐⭐ 새로 추가: 각 파일의 상세 분석 정보 표시
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 16px;">
                    <div style="font-weight: bold; color: var(--text-primary); margin-bottom: 12px;">
                        📋 Detailed File Analysis:
                    </div>
            `;
            
            for (const file of files) {
                try {
                    // dgit show 명령어로 파일 상세 정보 가져오기
                    const showResult = await window.electron.dgit.showFile(currentProject.path, file.name);
                    
                    if (showResult.success && showResult.output) {
                        terminalStatus.innerHTML += `
                            <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-tertiary); border-radius: 6px; border-left: 3px solid var(--accent-blue);">
                                <pre style="margin: 0; font-family: 'Consolas', 'Monaco', monospace; font-size: 0.85rem; color: var(--text-primary); white-space: pre-wrap; word-wrap: break-word;">${showResult.output}</pre>
                            </div>
                        `;
                    }
                } catch (error) {
                    console.error(`파일 분석 실패: ${file.name}`, error);
                }
            }
            
            terminalStatus.innerHTML += `</div>`;
            terminalStatus.scrollTop = terminalStatus.scrollHeight;
            
            // ⭐⭐ 수정: loadProjectData() 호출 제거 (터미널 내용 유지)
            // 파일 목록만 업데이트 (터미널 초기화 안 함)
            const fileListContainer = document.getElementById('fileList');
            if (fileListContainer) {
                // 파일 목록 렌더링
                const formattedFiles = files.map(file => ({
                    name: file.name,
                    type: file.type,
                    size: formatFileSize(file.size),
                    modified: formatDate(file.modified),
                    status: 'untracked',
                    path: file.path
                }));
                renderFiles(formattedFiles);
            }
            
            showToast(`스캔 완료: ${totalFiles}개 파일 분석 완료 (${totalSizeMB} MB)`, 'success');
            
        } else {
            // 파일이 없는 경우
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 16px; padding: 12px; background: var(--bg-secondary); border-left: 3px solid var(--accent-orange); border-radius: 4px;">
                    <div style="color: var(--text-secondary);">No design files found.</div>
                    <div style="margin-top: 8px; margin-left: 20px; color: var(--text-tertiary); font-size: 0.85rem;">
                        Supported formats: .ai, .psd, .sketch, .fig, .xd, .afdesign, .afphoto
                    </div>
                </div>
            `;
            terminalStatus.scrollTop = terminalStatus.scrollHeight;
            
            showToast('디자인 파일을 찾을 수 없습니다', 'warning');
        }
        
    } catch (error) {
        console.error('프로젝트 스캔 실패:', error);
        
        // 오류 메시지를 상태 터미널에 추가
        const terminalStatus = document.getElementById('terminalStatus');
        terminalStatus.innerHTML += `
            <div style="margin-bottom: 16px; padding: 12px; background: var(--bg-secondary); border-left: 3px solid var(--accent-red); border-radius: 4px;">
                <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
                    <span style="color: var(--accent-red); font-size: 1.2rem;">❌</span>
                    <span style="font-weight: bold; color: var(--accent-red);">Scan Error</span>
                </div>
                <div style="color: var(--text-secondary); font-size: 0.9rem;">
                    <span>[${new Date().toLocaleTimeString()}]</span>
                </div>
                <div style="margin-top: 8px; padding: 8px; background: var(--bg-tertiary); border-radius: 4px; font-family: monospace; font-size: 0.85rem; color: var(--accent-red);">
                    ${error.message}
                </div>
            </div>
        `;
        terminalStatus.scrollTop = terminalStatus.scrollHeight;
        
        showToast('프로젝트 스캔 중 오류가 발생했습니다', 'error');
    }
}

// Finder에서 보기
async function showInFinder() {
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    try {
        const result = await window.electron.showInFinder(currentProject.path);

        if (result.success) {
            showToast('Finder에서 프로젝트 폴더를 열었습니다', 'success');
        } else {
            showToast('Finder에서 열기 중 오류가 발생했습니다', 'error');
        }
    } catch (error) {
        console.error('Finder에서 보기 실패:', error);
        showToast('Finder에서 열기 중 오류가 발생했습니다', 'error');
    }
}

// 파일 타입별 아이콘 가져오기
function getFileIconForType(type) {
    const icons = {
        'psd': '🎨',
        'ai': '✏️',
        'sketch': '📐',
        'fig': '🎯',
        'xd': '💎',
        'png': '🖼️',
        'jpg': '🖼️',
        'jpeg': '🖼️',
        'gif': '🎞️',
        'svg': '🎨',
        'pdf': '📄',
        'txt': '📝',
        'md': '📖',
        'json': '⚙️',
        'js': '⚡',
        'css': '🎨',
        'html': '🌐'
    };
    return icons[type?.toLowerCase()] || '📄';
}
// 커밋 모달 표시
function showCommitModal() {
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    showModal('변경사항 커밋', '커밋 메시지를 입력하세요', `
        <div style="padding: 20px;">
            <div style="margin-bottom: 20px;">
                <h3 style="margin-bottom: 12px; color: var(--text-primary);">✍️ 커밋 메시지</h3>
                <p style="color: var(--text-secondary); line-height: 1.5; margin-bottom: 16px;">
                    변경사항에 대한 설명을 간결하게 작성해주세요.
                </p>
                <textarea id="commitMessage" class="input-field" placeholder="예: 홈페이지 UI 개선" style="width: 100%; height: 100px; resize: vertical;"></textarea>
            </div>
            <div style="display: flex; gap: 12px; justify-content: flex-end;">
                <button class="btn btn-secondary" onclick="closeModal()">취소</button>
                <button class="btn btn-primary" onclick="executeCommit()">
                    커밋 실행
                </button>
            </div>
        </div>
    `);

    // 모달이 열리면 textarea에 자동으로 포커스
    setTimeout(() => document.getElementById('commitMessage').focus(), 100);
}

// 커밋 실행
async function executeCommit() {
    const message = document.getElementById('commitMessage').value;

    if (!message.trim()) {
        showToast('커밋 메시지를 입력해주세요', 'warning');
        return;
    }

    closeModal();
    showToast('커밋을 진행합니다...', 'info');

    try {
        const result = await window.electron.dgit.commit(currentProject.path, message);

        if (result.success) {
            showToast('변경사항이 성공적으로 커밋되었습니다', 'success');
            
            // 상태 터미널에 성공 메시지 추가
            const terminalStatus = document.getElementById('terminalStatus');
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 8px; padding: 10px; background: var(--bg-secondary); border-left: 3px solid var(--accent-green); border-radius: 4px;">
                    <span style="color: var(--accent-green);">📌</span>
                    <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
                    <span style="margin-left: 8px; font-weight: bold; color: var(--text-primary);">커밋 완료</span>
                    <div style="margin-top: 6px; margin-left: 24px; color: var(--text-secondary);">${message}</div>
                </div>
            `;
            terminalStatus.scrollTop = terminalStatus.scrollHeight;

            // 프로젝트 데이터 전체 새로고침
            await loadProjectData();
        } else {
            showToast(`커밋 실패: ${result.error}`, 'error');
            
            // 상태 터미널에 오류 메시지 추가
            const terminalStatus = document.getElementById('terminalStatus');
            terminalStatus.innerHTML += `
                <div style="margin-bottom: 8px; padding: 10px; background: var(--bg-secondary); border-left: 3px solid var(--accent-red); border-radius: 4px;">
                    <span style="color: var(--accent-red);">❌</span>
                    <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
                    <span style="margin-left: 8px; font-weight: bold; color: var(--accent-red);">커밋 실패</span>
                    <div style="margin-top: 6px; margin-left: 24px; color: var(--text-secondary); font-family: monospace;">${result.error}</div>
                </div>
            `;
            terminalStatus.scrollTop = terminalStatus.scrollHeight;
        }
    } catch (error) {
        console.error('커밋 실행 오류:', error);
        showToast('커밋 중 오류가 발생했습니다', 'error');
    }
}
