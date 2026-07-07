(function() {
    if (/iPhone|iPad|iPod/i.test(navigator.userAgent)) {
        document.addEventListener('gesturestart', function(e) {
            e.preventDefault();
        }, { passive: false });

        var lastTouchEnd = 0;
        document.addEventListener('touchend', function(e) {
            var now = Date.now();
            if (now - lastTouchEnd <= 300) {
                e.preventDefault();
            }
            lastTouchEnd = now;
        }, { passive: false });
    }
})();
