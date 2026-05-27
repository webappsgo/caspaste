// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Mobile navigation - optional JS enhancements (close-on-outside-click, Escape).
// The toggle itself is pure CSS via the #nav-toggle-state checkbox.
(function() {
	var navCheckbox = document.getElementById('nav-toggle-state');
	var navLabel = document.querySelector('label[for="nav-toggle-state"]');

	if (navCheckbox) {
		// Close menu when clicking outside the nav
		document.addEventListener('click', function(e) {
			if (navCheckbox.checked && !e.target.closest('header nav')) {
				navCheckbox.checked = false;
			}
		});

		// Close menu on Escape key and return focus to the toggle label
		document.addEventListener('keydown', function(e) {
			if (e.key === 'Escape' && navCheckbox.checked) {
				navCheckbox.checked = false;
				if (navLabel) {
					navLabel.focus();
				}
			}
		});

		// Close menu when a nav link is activated
		document.querySelectorAll('.nav-links a, .nav-links button').forEach(function(el) {
			el.addEventListener('click', function() {
				navCheckbox.checked = false;
			});
		});
	}

	// PWA Service Worker registration
	if ('serviceWorker' in navigator) {
		navigator.serviceWorker.register('/sw.js').catch(function(error) {
			console.log('Service Worker registration failed:', error);
		});
	}
})();
