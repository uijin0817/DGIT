// 유틸리티 함수들

// 파일 크기 포맷팅
function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// 날짜 포맷팅
function formatDate(dateInput) {
    let date;

    if (typeof dateInput === 'string') {
        date = new Date(dateInput);
    } else if (dateInput instanceof Date) {
        date = dateInput;
    } else {
        return 'Invalid Date';
    }

    if (isNaN(date.getTime())) {
        return 'Invalid Date';
    }

    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return '방금 전';
    if (diffMins < 60) return `${diffMins}분 전`;
    if (diffHours < 24) return `${diffHours}시간 전`;
    if (diffDays < 7) return `${diffDays}일 전`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)}주 전`;
    if (diffDays < 365) return `${Math.floor(diffDays / 30)}개월 전`;

    return date.toLocaleDateString('ko-KR', {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    });
}

// 상대적 시간 포맷팅 (더 정확한 버전)
function formatRelativeTime(date) {
    const now = new Date();
    const diffMs = now - date;
    const diffSecs = Math.floor(diffMs / 1000);
    const diffMins = Math.floor(diffSecs / 60);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);
    const diffWeeks = Math.floor(diffDays / 7);
    const diffMonths = Math.floor(diffDays / 30);
    const diffYears = Math.floor(diffDays / 365);

    if (diffSecs < 60) return '방금 전';
    if (diffMins < 60) return `${diffMins}분 전`;
    if (diffHours < 24) return `${diffHours}시간 전`;
    if (diffDays < 7) return `${diffDays}일 전`;
    if (diffWeeks < 4) return `${diffWeeks}주 전`;
    if (diffMonths < 12) return `${diffMonths}개월 전`;
    return `${diffYears}년 전`;
}

// 절대 시간 포맷팅
function formatAbsoluteTime(date, options = {}) {
    const defaultOptions = {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        ...options
    };

    return new Intl.DateTimeFormat('ko-KR', defaultOptions).format(date);
}

// 문자열 자르기 (말줄임표 추가)
function truncateString(str, maxLength, suffix = '...') {
    if (!str || str.length <= maxLength) return str;
    return str.substring(0, maxLength - suffix.length) + suffix;
}

// 파일 확장자 추출
function getFileExtension(filename) {
    if (!filename || typeof filename !== 'string') return '';
    const lastDotIndex = filename.lastIndexOf('.');
    return lastDotIndex === -1 ? '' : filename.substring(lastDotIndex + 1).toLowerCase();
}

// 파일명에서 확장자 제거
function removeFileExtension(filename) {
    if (!filename || typeof filename !== 'string') return '';
    const lastDotIndex = filename.lastIndexOf('.');
    return lastDotIndex === -1 ? filename : filename.substring(0, lastDotIndex);
}

// 숫자 포맷팅 (천 단위 콤마)
function formatNumber(num) {
    if (typeof num !== 'number') return '0';
    return new Intl.NumberFormat('ko-KR').format(num);
}

// 퍼센트 포맷팅
function formatPercent(value, total, decimals = 1) {
    if (total === 0) return '0%';
    const percent = (value / total) * 100;
    return `${percent.toFixed(decimals)}%`;
}

// 색상 유틸리티
const ColorUtils = {
    // HEX를 RGB로 변환
};

// 디바운스 함수
function debounce(func, wait, immediate = false) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            timeout = null;
            if (!immediate) func(...args);
        };
        const callNow = immediate && !timeout;
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
        if (callNow) func(...args);
    };
}

// 스로틀 함수
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// 깊은 복사
function deepClone(obj) {
    if (obj === null || typeof obj !== 'object') return obj;
    if (obj instanceof Date) return new Date(obj.getTime());
    if (obj instanceof Array) return obj.map(item => deepClone(item));
    if (typeof obj === 'object') {
        const clonedObj = {};
        for (const key in obj) {
            if (Object.prototype.hasOwnProperty.call(obj, key)) {
                clonedObj[key] = deepClone(obj[key]);
            }
        }
        return clonedObj;
    }
}

// 객체 비교
function isEqual(obj1, obj2) {
    if (obj1 === obj2) return true;
    if (obj1 == null || obj2 == null) return false;
    if (typeof obj1 !== typeof obj2) return false;

    if (typeof obj1 !== 'object') return obj1 === obj2;

    const keys1 = Object.keys(obj1);
    const keys2 = Object.keys(obj2);

    if (keys1.length !== keys2.length) return false;

    for (let key of keys1) {
        if (!keys2.includes(key)) return false;
        if (!isEqual(obj1[key], obj2[key])) return false;
    }

    return true;
}

// 배열에서 중복 제거
function uniqueArray(arr, key = null) {
    if (key) {
        const seen = new Set();
        return arr.filter(item => {
            const value = item[key];
            if (seen.has(value)) {
                return false;
            }
            seen.add(value);
            return true;
        });
    }
    return [...new Set(arr)];
}

// 배열 섞기
function shuffleArray(arr) {
    const shuffled = [...arr];
    for (let i = shuffled.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    return shuffled;
}

// 랜덤 문자열 생성
function generateRandomString(length = 8, chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789') {
    let result = '';
    for (let i = 0; i < length; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
}

// UUID 생성 (간단한 버전)
function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

// 로컬 스토리지 유틸리티
const StorageUtils = {
    set(key, value) {
        try {
            localStorage.setItem(key, JSON.stringify(value));
            return true;
        } catch (error) {
            console.error('localStorage set error:', error);
            return false;
        }
    },

    get(key, defaultValue = null) {
        try {
            const item = localStorage.getItem(key);
            return item ? JSON.parse(item) : defaultValue;
        } catch (error) {
            console.error('localStorage get error:', error);
            return defaultValue;
        }
    },
    clear() {
        try {
            localStorage.clear();
            return true;
        } catch (error) {
            console.error('localStorage clear error:', error);
            return false;
        }
    }
};

// URL 유틸리티
const UrlUtils = {
    // 쿼리 파라미터를 객체로 변환
// 객체를 쿼리 스트링으로 변환
// URL 유효성 검사
};

// 에러 처리 유틸리티
const ErrorUtils = {
    // 에러 메시지 정규화
};

// 성능 측정 유틸리티
const PerformanceUtils = {
    // 함수 실행 시간 측정
// 비동기 함수 실행 시간 측정
};

// 환경 감지 유틸리티
const EnvUtils = {
    // 운영체제 감지
// 브라우저 감지
// 모바일 기기 감지
// 터치 지원 감지
};

// 전역에서 사용할 수 있도록 export
window.Utils = {
    formatFileSize,
    formatDate,
    formatRelativeTime,
    formatAbsoluteTime,
    truncateString,
    getFileExtension,
    removeFileExtension,
    formatNumber,
    formatPercent,
    ColorUtils,
    debounce,
    throttle,
    deepClone,
    isEqual,
    uniqueArray,
    shuffleArray,
    generateRandomString,
    generateUUID,
    StorageUtils,
    UrlUtils,
    ErrorUtils,
    PerformanceUtils,
    EnvUtils
};