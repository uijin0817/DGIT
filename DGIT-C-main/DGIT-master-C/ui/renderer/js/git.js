// Git 관련 함수들

// ⭐⭐ 수정: 변경사항 커밋 - 간소화 및 개선
async function commitChanges() {
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    try {
        // 커밋 메시지 입력 모달 바로 표시
        showModal('변경사항 커밋', '커밋 메시지를 입력하세요', `
            <div style="padding: 20px;">
                <div style="margin-bottom: 20px;">
                    <h3 style="margin-bottom: 12px; color: var(--text-primary);">✍️ 커밋 메시지</h3>
                    <p style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 16px;">
                        변경사항을 설명하는 메시지를 입력해주세요.
                    </p>
                    <textarea
                        id="commitMessageInput"
                        placeholder="예: 디자인 파일 업데이트, 새로운 아이콘 추가 등..."
                        style="width: 100%; min-height: 100px; padding: 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-secondary); color: var(--text-primary); font-size: 0.95rem; resize: vertical;"
                    ></textarea>
                </div>
                <div style="display: flex; gap: 12px; justify-content: flex-end;">
                    <button class="btn btn-secondary" onclick="closeModal()">취소</button>
                    <button class="btn btn-primary" onclick="executeCommit()">커밋 실행</button>
                </div>
            </div>
        `);
        
        // 텍스트 영역에 포커스
        setTimeout(() => {
            const textarea = document.getElementById('commitMessageInput');
            if (textarea) {
                textarea.focus();
            }
        }, 100);
        
    } catch (error) {
        console.error('커밋 준비 실패:', error);
        showToast('커밋 준비 중 오류가 발생했습니다', 'error');
    }
}

// ⭐⭐ 새로 추가: 커밋 실행 함수
async function executeCommit() {
    const messageInput = document.getElementById('commitMessageInput');
    const message = messageInput ? messageInput.value.trim() : '';
    
    if (!message) {
        showToast('커밋 메시지를 입력해주세요', 'warning');
        return;
    }
    
    closeModal();
    
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'error');
        return;
    }

    try {
        // 커밋 진행 모달 표시
        showModal('커밋 진행 중', '', `
            <div style="padding: 30px; text-align: center;">
                <div style="margin-bottom: 20px;">
                    <div class="loading-spinner" style="margin: 0 auto;"></div>
                </div>
                <div id="commitProgressText" style="color: var(--text-secondary); font-size: 1rem;">
                    커밋을 시작합니다...
                </div>
            </div>
        `);

        // 로그 탭으로 전환
        showTerminalTab('log');
        
        // 터미널에 로그 추가
        const terminalLog = document.getElementById('terminalLog');
        terminalLog.innerHTML += `
            <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-tertiary); border-radius: 6px;">
                <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
                    <span style="color: var(--accent-blue); font-size: 1.2rem;">💾</span>
                    <span style="font-weight: bold; color: var(--text-primary);">커밋 시작</span>
                </div>
                <div style="color: var(--text-secondary); font-size: 0.9rem;">
                    <span>[${new Date().toLocaleTimeString()}]</span>
                    <span style="margin-left: 8px;">메시지: "${message}"</span>
                </div>
            </div>
        `;
        terminalLog.scrollTop = terminalLog.scrollHeight;

        // 1. 모든 파일 추가
        document.getElementById('commitProgressText').textContent = '파일을 스테이징 영역에 추가하는 중...';
        
        const addResult = await window.electron.dgit.add(currentProject.path);
        
        if (!addResult.success) {
            throw new Error(`파일 추가 실패: ${addResult.error}`);
        }

        terminalLog.innerHTML += `
            <div style="margin-bottom: 8px;">
                <span style="color: var(--accent-green);">✅</span>
                <span style="color: var(--text-secondary);">[${new Date().toLocaleTimeString()}]</span>
                <span style="margin-left: 8px;">파일 추가 완료</span>
            </div>
        `;
        terminalLog.scrollTop = terminalLog.scrollHeight;

        // 2. 커밋 실행
        document.getElementById('commitProgressText').textContent = '커밋을 생성하는 중...';
        
        const commitResult = await window.electron.dgit.commit(currentProject.path, message);

        if (commitResult.success) {
            // 성공 로그 추가
            terminalLog.innerHTML += `
                <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-tertiary); border-left: 3px solid var(--accent-green); border-radius: 4px;">
                    <div style="display: flex; align-items: center; gap: 8px;">
                        <span style="color: var(--accent-green); font-size: 1.2rem;">✅</span>
                        <span style="font-weight: bold; color: var(--accent-green);">커밋 완료</span>
                    </div>
                    <div style="color: var(--text-secondary); font-size: 0.9rem; margin-top: 4px;">
                        <span>[${new Date().toLocaleTimeString()}]</span>
                        <span style="margin-left: 8px;">메시지: "${message}"</span>
                    </div>
                </div>
            `;
            terminalLog.scrollTop = terminalLog.scrollHeight;

            // 모달 닫고 성공 메시지
            setTimeout(async () => {
                closeModal();
                showToast('커밋이 성공적으로 완료되었습니다', 'success');
                
                // 프로젝트 데이터 새로고침 (커밋 히스토리 업데이트)
                await loadProjectData();
            }, 1000);

        } else {
            throw new Error(commitResult.error || '커밋 실패');
        }
        
    } catch (error) {
        console.error('커밋 실행 실패:', error);

        // 오류 로그 추가
        const terminalLog = document.getElementById('terminalLog');
        terminalLog.innerHTML += `
            <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-secondary); border-left: 3px solid var(--accent-red); border-radius: 4px;">
                <div style="display: flex; align-items: center; gap: 8px;">
                    <span style="color: var(--accent-red); font-size: 1.2rem;">❌</span>
                    <span style="font-weight: bold; color: var(--accent-red);">커밋 실패</span>
                </div>
                <div style="color: var(--text-secondary); font-size: 0.9rem; margin-top: 4px;">
                    <span>[${new Date().toLocaleTimeString()}]</span>
                </div>
                <div style="margin-top: 8px; padding: 8px; background: var(--bg-tertiary); border-radius: 4px; font-family: monospace; font-size: 0.85rem; color: var(--accent-red);">
                    ${error.message}
                </div>
            </div>
        `;
        terminalLog.scrollTop = terminalLog.scrollHeight;

        setTimeout(() => {
            closeModal();
            showToast('커밋 중 오류가 발생했습니다', 'error');
        }, 1500);
    }
}

// ⭐⭐ 수정: 파일 복원 - DGit restore 명령어 사용
async function restoreFiles() {
    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'warning');
        return;
    }

    try {
        // 커밋 히스토리 가져오기
        const logResult = await window.electron.dgit.log(currentProject.path, 10);
        
        if (!logResult.success || !logResult.output) {
            showToast('커밋 히스토리를 불러올 수 없습니다', 'error');
            return;
        }

        const commits = parseCommitLog(logResult.output);
        
        if (commits.length === 0) {
            showToast('복원할 커밋이 없습니다', 'warning');
            return;
        }

        // 복원할 커밋 선택 모달
        showModal('파일 복원', '복원할 버전을 선택하세요', `
            <div style="padding: 20px;">
                <p style="margin-bottom: 20px; color: var(--text-secondary);">
                    선택한 버전의 모든 파일이 복원됩니다.<br>
                    <strong style="color: var(--accent-orange);">주의: 현재 변경사항은 모두 사라집니다.</strong>
                </p>
                
                <div style="max-height: 400px; overflow-y: auto; margin-bottom: 20px;">
                    ${commits.map(commit => `
                        <div class="commit-item-selectable" onclick="selectCommitForRestore('${commit.version}', '${commit.hash}', '${commit.message.replace(/'/g, "\\'")}')">
                            <div style="display: flex; justify-content: space-between; align-items: center; padding: 12px; background: var(--bg-secondary); border-radius: 6px; cursor: pointer; transition: all 0.2s;" onmouseover="this.style.background='var(--bg-tertiary)'" onmouseout="this.style.background='var(--bg-secondary)'">
                                <div>
                                    <div style="font-weight: bold; color: var(--text-primary); margin-bottom: 4px;">
                                        v${commit.version} - ${commit.message}
                                    </div>
                                    <div style="font-size: 0.85rem; color: var(--text-secondary);">
                                        ${commit.hash} • ${commit.date} • ${commit.files} 파일
                                    </div>
                                </div>
                                <div style="color: var(--accent-blue);">
                                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                        <path d="M9 18l6-6-6-6"/>
                                    </svg>
                                </div>
                            </div>
                        </div>
                    `).join('')}
                </div>
                
                <div style="display: flex; gap: 12px; justify-content: center;">
                    <button class="btn btn-secondary" onclick="closeModal()">취소</button>
                </div>
            </div>
        `);
        
    } catch (error) {
        console.error('복원 준비 실패:', error);
        showToast('복원 준비 중 오류가 발생했습니다', 'error');
    }
}

// ⭐⭐ 새로 추가: 복원할 커밋 선택
function selectCommitForRestore(version, hash, message) {
    closeModal();
    
    showModal('복원 확인', '정말 복원하시겠습니까?', `
        <div style="padding: 20px; text-align: center;">
            <div style="margin-bottom: 20px;">
                <div style="font-size: 2.5rem; margin-bottom: 12px;">⚠️</div>
                <h3 style="margin-bottom: 12px; color: var(--text-primary);">버전 ${version}로 복원</h3>
                <p style="color: var(--text-secondary); margin-bottom: 8px;">
                    "${message}"
                </p>
                <p style="color: var(--accent-orange); font-size: 0.9rem;">
                    현재 작업 중인 모든 변경사항이 사라집니다.
                </p>
            </div>
            <div style="display: flex; gap: 12px; justify-content: center;">
                <button class="btn btn-secondary" onclick="closeModal()">취소</button>
                <button class="btn btn-danger" onclick="performRestoreToVersion('${version}')" style="background: #ff3b30 !important;">
                    복원 실행
                </button>
            </div>
        </div>
    `);
}

// ⭐⭐ 수정: 버전으로 복원 실행
async function performRestoreToVersion(version) {
    closeModal();

    if (!currentProject) {
        showToast('프로젝트가 선택되지 않았습니다', 'error');
        return;
    }

    try {
        // 복원 진행 모달 표시
        showModal('복원 진행 중', '', `
            <div style="padding: 30px; text-align: center;">
                <div style="margin-bottom: 20px;">
                    <div class="loading-spinner" style="margin: 0 auto;"></div>
                </div>
                <div id="restoreProgressText" style="color: var(--text-secondary); font-size: 1rem;">
                    버전 ${version}로 복원 중...
                </div>
            </div>
        `);

        // 로그 탭으로 전환
        showTerminalTab('log');
        
        // 터미널에 로그 추가
        const terminalLog = document.getElementById('terminalLog');
        terminalLog.innerHTML += `
            <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-tertiary); border-radius: 6px;">
                <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
                    <span style="color: var(--accent-orange); font-size: 1.2rem;">🔄</span>
                    <span style="font-weight: bold; color: var(--text-primary);">파일 복원 시작</span>
                </div>
                <div style="color: var(--text-secondary); font-size: 0.9rem;">
                    <span>[${new Date().toLocaleTimeString()}]</span>
                    <span style="margin-left: 8px;">버전: v${version}</span>
                </div>
            </div>
        `;
        terminalLog.scrollTop = terminalLog.scrollHeight;

        // DGit restore 명령 실행
        const restoreResult = await window.electron.dgit.restore(currentProject.path, version, []);

        if (restoreResult.success) {
            // 성공 로그 추가
            terminalLog.innerHTML += `
                <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-tertiary); border-left: 3px solid var(--accent-green); border-radius: 4px;">
                    <div style="display: flex; align-items: center; gap: 8px;">
                        <span style="color: var(--accent-green); font-size: 1.2rem;">✅</span>
                        <span style="font-weight: bold; color: var(--accent-green);">복원 완료</span>
                    </div>
                    <div style="color: var(--text-secondary); font-size: 0.9rem; margin-top: 4px;">
                        <span>[${new Date().toLocaleTimeString()}]</span>
                        <span style="margin-left: 8px;">버전 v${version}로 성공적으로 복원되었습니다</span>
                    </div>
                </div>
            `;
            terminalLog.scrollTop = terminalLog.scrollHeight;

            // 모달 닫고 성공 메시지
            setTimeout(async () => {
                closeModal();
                showToast(`버전 ${version}로 성공적으로 복원되었습니다`, 'success');
                
                // 프로젝트 데이터 새로고침
                await loadProjectData();
            }, 1000);

        } else {
            throw new Error(restoreResult.error || '복원 실패');
        }
        
    } catch (error) {
        console.error('복원 실행 실패:', error);

        // 오류 로그 추가
        const terminalLog = document.getElementById('terminalLog');
        terminalLog.innerHTML += `
            <div style="margin-bottom: 12px; padding: 12px; background: var(--bg-secondary); border-left: 3px solid var(--accent-red); border-radius: 4px;">
                <div style="display: flex; align-items: center; gap: 8px;">
                    <span style="color: var(--accent-red); font-size: 1.2rem;">❌</span>
                    <span style="font-weight: bold; color: var(--accent-red);">복원 실패</span>
                </div>
                <div style="color: var(--text-secondary); font-size: 0.9rem; margin-top: 4px;">
                    <span>[${new Date().toLocaleTimeString()}]</span>
                </div>
                <div style="margin-top: 8px; padding: 8px; background: var(--bg-tertiary); border-radius: 4px; font-family: monospace; font-size: 0.85rem; color: var(--accent-red);">
                    ${error.message}
                </div>
            </div>
        `;
        terminalLog.scrollTop = terminalLog.scrollHeight;

        setTimeout(() => {
            closeModal();
            showToast('복원 중 오류가 발생했습니다', 'error');
        }, 1500);
    }
}

// Git 상태 출력 파싱
function parseGitStatus(output) {
    const statusMap = {};
    const lines = output.split('\n');

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;

        // DGit 상태 출력 형식 파싱
        const match = trimmed.match(/^([MAD?!])\s+(.+)$/);
        if (match) {
            const [, statusCode, filename] = match;
            const status = getGitStatusText(statusCode);
            statusMap[filename] = status;
        }
    }

    return statusMap;
}

// Git 상태 코드를 텍스트로 변환
function getGitStatusText(statusCode) {
    const statusMap = {
        'M': 'modified',
        'A': 'added',
        'D': 'deleted',
        'R': 'renamed',
        'C': 'copied',
        'U': 'updated',
        'T': 'untracked',
        '!': 'ignored'
    };
    return statusMap[statusCode] || 'unknown';
}

// 커밋 로그 파싱
function parseCommitLog(output) {
    const commits = [];
    if (!output || !output.trim()) return commits;

    const lines = output.split('\n');
    let currentCommit = null;

    for (const line of lines) {
        if (line.startsWith('commit ')) {
            if (currentCommit) {
                commits.push(currentCommit);
            }
            // DGit 형식: "commit 4ea7d8384946 (v2)"
            const commitMatch = line.match(/commit\s+([a-f0-9]+)\s*\(([^)]+)\)/);
            if (commitMatch) {
                currentCommit = {
                    hash: commitMatch[1].substring(0, 7),
                    version: commitMatch[2],
                    message: '',
                    author: '',
                    date: '',
                    files: 0
                };
            }
        } else if (line.startsWith('Author: ') && currentCommit) {
            currentCommit.author = line.substring(8).trim();
        } else if (line.startsWith('Date: ') && currentCommit) {
            const dateStr = line.substring(6).trim();
            currentCommit.date = formatCommitDate(dateStr);
        } else if (line.startsWith('    ') && currentCommit && !currentCommit.message) {
            // 커밋 메시지는 4개 공백으로 들여쓰기됨
            currentCommit.message = line.trim();
        } else if (line.startsWith('    Files: ') && currentCommit) {
            const filesMatch = line.match(/Files:\s+(\d+)/);
            if (filesMatch) {
                currentCommit.files = parseInt(filesMatch[1]);
            }
        }
    }

    if (currentCommit) {
        commits.push(currentCommit);
    }

    return commits;
}

// 커밋 날짜 포맷팅
function formatCommitDate(dateStr) {
    try {
        const date = new Date(dateStr);
        return formatDate(date);
    } catch (error) {
        return dateStr;
    }
}