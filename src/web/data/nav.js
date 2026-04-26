// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Mobile navigation toggle
(function() {
	var navToggle = document.getElementById('js-nav-toggle');
	var navLinks = document.getElementById('js-nav-links');

	if (navToggle && navLinks) {
		navToggle.addEventListener('click', function() {
			var isOpen = navLinks.classList.toggle('open');
			navToggle.classList.toggle('active');
			navToggle.setAttribute('aria-expanded', isOpen);
		});

		// Close menu when clicking outside
		document.addEventListener('click', function(e) {
			if (!navToggle.contains(e.target) && !navLinks.contains(e.target)) {
				navLinks.classList.remove('open');
				navToggle.classList.remove('active');
				navToggle.setAttribute('aria-expanded', 'false');
			}
		});

		// Close menu when pressing Escape
		document.addEventListener('keydown', function(e) {
			if (e.key === 'Escape' && navLinks.classList.contains('open')) {
				navLinks.classList.remove('open');
				navToggle.classList.remove('active');
				navToggle.setAttribute('aria-expanded', 'false');
				navToggle.focus();
			}
		});
	}

	// PWA Service Worker registration
	if ('serviceWorker' in navigator) {
		navigator.serviceWorker.register('/sw.js').catch(function(error) {
			console.log('Service Worker registration failed:', error);
		});
	}
})();
