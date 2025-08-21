/**
 * SharePoint Audit - Application JavaScript
 * Minimal JavaScript for HTMX enhancements
 */

// Initialize application when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Add global keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Ctrl/Cmd + K to focus search
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            const searchInput = document.querySelector('input[type="search"]');
            if (searchInput) {
                searchInput.focus();
            }
        }
        
        // Escape to clear search
        if (e.key === 'Escape') {
            const searchInput = document.querySelector('input[type="search"]');
            if (searchInput && searchInput === document.activeElement) {
                searchInput.value = '';
                // Trigger HTMX search clear
                htmx.trigger(searchInput, 'input');
            }
        }
    });
});