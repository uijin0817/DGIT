/* globals.js - ESLint 전역 함수 선언 */

/* eslint-disable no-unused-vars */

// app.js 함수들
let initializeApp;
let setupTerminalResizer;
let loadAppConfig;
let updateNotificationUI;
let loadRecentProjects;
let selectNewProject;
let openCurrentProject;
let showRecentProjects;
let openRecentProject;
let goHome;
let showContent;
let showTerminalTab;
let toggleNotifications;
let saveNotificationSettings;
let toggleTerminalCollapse; 

// project.js 함수들
let checkDGitAvailability;
let checkIfRepository;
let selectProjectWithoutDGit;
let openProjectWithoutDGit;
let loadProjectFilesOnly;
let initializeRepository;
let initializeCurrentProject;
let executeInitialization;
let checkRepositoryStatusAndShowInitButton;
let openProject;
let showDGitInitPrompt;
let openProjectDirectly;
let loadProjectData;
let loadProjectFiles;
let loadCommitHistory;
let updateProjectStatus;
let updateFileStatuses;
let changeProject;
let scanProject;
let showInFinder;

// git.js 함수들
let commitChanges;
let performCommit;
let addAllFiles;
let restoreFiles;
let performRestore;
let restoreToCommit;
let performRestoreToCommit;
let parseGitStatus;
let getGitStatusText;
let parseCommitLog;
let formatCommitDate;

// ui.js 함수들
let renderFiles;
let renderCommits;
let isPreviewableImage;
let showImagePreview;
let closeImagePreview;
let selectFile;
let showModal;
let closeModal;
let confirmModal;
let showToast; // ⭐ 추가됨
let updateTerminalStatus;
let viewCommit;
let viewCommitDiff;
let getFileIcon;
let getStatusColor;
let showLoadingSpinner;
let hideLoadingSpinner;
let addAnimation;
let triggerHapticFeedback;
let showEmptyState;
let showErrorState;
let showProgressBar;
let updateProgressBar;
let showCircularProgress;
let showStepProgress;
let showRealtimeProgress;
let showFileProgress;
let formatTime;
let showContextMenu;
let setupDropZone;
let showKeyboardShortcuts;
let toggleFullscreen;
let toggleDevTools;

// 전역 변수들
let currentProject;
let activeContent;
let notificationsEnabled;
let recentProjectsList;
let currentModalCallback;
let isTerminalCollapsed; 

// ⭐⭐ 수정: window 객체에 노출된 변수 및 객체를 전역에서 사용될 수 있도록 추가
let window; 

// 유틸리티 객체들
let ColorUtils;
let StorageUtils;
let UrlUtils;
let ErrorUtils;
let PerformanceUtils;
let EnvUtils;
let Utils;