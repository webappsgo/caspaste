/**
 * Toast notification system per AI.md PART 16
 * Provides non-blocking notifications for user feedback
 */

(function() {
    'use strict';

    // Toast configuration
    const CONFIG = {
        maxVisible: 5,
        defaultDuration: 3000,
        warningDuration: 5000,
        position: 'top-right',
        animationDuration: 300
    };

    // Toast types with their properties
    const TOAST_TYPES = {
        success: { icon: '\u2713', autoDismiss: true, duration: CONFIG.defaultDuration },
        error: { icon: '\u2717', autoDismiss: false, duration: 0 },
        warning: { icon: '\u26A0', autoDismiss: true, duration: CONFIG.warningDuration },
        info: { icon: '\u2139', autoDismiss: true, duration: CONFIG.defaultDuration }
    };

    // Active toasts
    let toasts = [];
    let toastQueue = [];
    let toastIdCounter = 0;
    let container = null;

    /**
     * Initialize the toast container
     */
    function initContainer() {
        if (container) return container;

        container = document.getElementById('toast-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'toast-container';
            container.setAttribute('aria-label', 'Notifications');
            document.body.appendChild(container);
        }
        return container;
    }

    /**
     * Create a toast element
     * @param {string} message - Toast message
     * @param {string} type - Toast type (success, error, warning, info)
     * @param {number} id - Toast ID
     * @param {number} duration - Auto-dismiss duration in ms
     * @returns {HTMLElement} Toast element
     */
    function createToastElement(message, type, id, duration) {
        const typeConfig = TOAST_TYPES[type] || TOAST_TYPES.info;

        const toast = document.createElement('div');
        toast.className = 'toast toast-' + type;
        toast.setAttribute('role', 'alert');
        toast.setAttribute('aria-live', 'polite');
        toast.setAttribute('data-toast-id', id);

        // Icon
        const icon = document.createElement('span');
        icon.className = 'toast-icon';
        icon.textContent = typeConfig.icon;
        toast.appendChild(icon);

        // Message
        const msg = document.createElement('span');
        msg.className = 'toast-message';
        msg.textContent = message;
        toast.appendChild(msg);

        // Close button
        const closeBtn = document.createElement('button');
        closeBtn.className = 'toast-close';
        closeBtn.setAttribute('aria-label', 'Dismiss');
        closeBtn.innerHTML = '&times;';
        closeBtn.onclick = function() { dismissToast(id); };
        toast.appendChild(closeBtn);

        // Progress bar for auto-dismiss
        if (duration > 0) {
            const progress = document.createElement('div');
            progress.className = 'toast-progress';
            progress.style.animationDuration = duration + 'ms';
            toast.appendChild(progress);
        }

        // Click to dismiss (except on close button)
        toast.onclick = function(e) {
            if (e.target !== closeBtn) {
                dismissToast(id);
            }
        };

        return toast;
    }

    /**
     * Show a toast notification
     * @param {string} message - Toast message
     * @param {string} type - Toast type (success, error, warning, info)
     * @param {number} [duration] - Optional custom duration in ms
     * @returns {number} Toast ID
     */
    function showToast(message, type, duration) {
        type = type || 'info';
        const typeConfig = TOAST_TYPES[type] || TOAST_TYPES.info;

        // Determine duration
        if (duration === undefined) {
            duration = typeConfig.autoDismiss ? typeConfig.duration : 0;
        }

        const id = ++toastIdCounter;
        const toastData = { id: id, message: message, type: type, duration: duration };

        // Queue if at max capacity
        if (toasts.length >= CONFIG.maxVisible) {
            toastQueue.push(toastData);
            return id;
        }

        displayToast(toastData);
        return id;
    }

    /**
     * Display a toast (internal)
     * @param {Object} toastData - Toast data object
     */
    function displayToast(toastData) {
        initContainer();

        const element = createToastElement(
            toastData.message,
            toastData.type,
            toastData.id,
            toastData.duration
        );

        // Insert at top (newest first)
        container.insertBefore(element, container.firstChild);

        // Track toast
        const toast = {
            id: toastData.id,
            element: element,
            duration: toastData.duration,
            timeoutId: null,
            paused: false,
            remaining: toastData.duration
        };
        toasts.push(toast);

        // Set up auto-dismiss
        if (toastData.duration > 0) {
            toast.startTime = Date.now();
            toast.timeoutId = setTimeout(function() {
                dismissToast(toastData.id);
            }, toastData.duration);

            // Pause on hover
            element.addEventListener('mouseenter', function() {
                pauseToast(toast);
            });
            element.addEventListener('mouseleave', function() {
                resumeToast(toast);
            });
        }
    }

    /**
     * Pause a toast's countdown
     * @param {Object} toast - Toast object
     */
    function pauseToast(toast) {
        if (toast.paused || !toast.timeoutId) return;

        clearTimeout(toast.timeoutId);
        toast.paused = true;
        toast.remaining -= (Date.now() - toast.startTime);

        // Pause progress animation
        const progress = toast.element.querySelector('.toast-progress');
        if (progress) {
            progress.classList.add('toast-progress-paused');
            progress.classList.remove('toast-progress-running');
        }
    }

    /**
     * Resume a toast's countdown
     * @param {Object} toast - Toast object
     */
    function resumeToast(toast) {
        if (!toast.paused) return;

        toast.paused = false;
        toast.startTime = Date.now();
        toast.timeoutId = setTimeout(function() {
            dismissToast(toast.id);
        }, toast.remaining);

        // Resume progress animation
        const progress = toast.element.querySelector('.toast-progress');
        if (progress) {
            progress.classList.remove('toast-progress-paused');
            progress.classList.add('toast-progress-running');
        }
    }

    /**
     * Dismiss a specific toast
     * @param {number} id - Toast ID
     */
    function dismissToast(id) {
        const index = toasts.findIndex(function(t) { return t.id === id; });
        if (index === -1) return;

        const toast = toasts[index];

        // Clear timeout
        if (toast.timeoutId) {
            clearTimeout(toast.timeoutId);
        }

        // Animate out
        toast.element.classList.add('toast-dismissing');

        setTimeout(function() {
            if (toast.element.parentNode) {
                toast.element.parentNode.removeChild(toast.element);
            }
            toasts.splice(index, 1);

            // Show next queued toast
            if (toastQueue.length > 0 && toasts.length < CONFIG.maxVisible) {
                displayToast(toastQueue.shift());
            }
        }, CONFIG.animationDuration);
    }

    /**
     * Dismiss all toasts
     */
    function dismissAllToasts() {
        // Clear queue
        toastQueue = [];

        // Dismiss all active toasts
        var ids = toasts.map(function(t) { return t.id; });
        ids.forEach(dismissToast);
    }

    /**
     * Handle escape key to dismiss topmost toast
     */
    function handleKeydown(e) {
        if (e.key === 'Escape' && toasts.length > 0) {
            dismissToast(toasts[toasts.length - 1].id);
        }
    }

    // Set up event listeners
    document.addEventListener('keydown', handleKeydown);

    // Expose public API
    window.showToast = showToast;
    window.dismissToast = dismissToast;
    window.dismissAllToasts = dismissAllToasts;

})();
