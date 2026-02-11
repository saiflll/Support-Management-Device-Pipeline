function appData() {
    console.log("forming.js initialized");
    return {
        activeTab: 'all',
        prefixes: [],
        statusFilter: 'all',
        sortBy: 'newest',
        startDate: '',
        endDate: '',

        init() {
            this.loadPrefixes();
            this.loadData();
            this.refreshSummary();
            console.log("Alpine init complete");
        },

        async loadPrefixes() {
            try {
                const r = await fetch('/prefixes');
                this.prefixes = await r.json();
            } catch (e) {
                console.error('Error loading prefixes:', e);
            }
        },

        loadData() {
            const params = new URLSearchParams({
                prefix: this.activeTab,
                status: this.statusFilter,
                sort: this.sortBy,
                start_date: this.startDate,
                end_date: this.endDate
            });

            let url = '/data-list?' + params.toString();
            if (this.startDate || this.endDate) {
                url = '/data-by-date?' + params.toString();
            } else if (this.activeTab !== 'all') {
                url = '/data-by-prefix?' + params.toString();
            }

            htmx.ajax('GET', url, {
                target: '#data-tbody',
                swap: 'innerHTML'
            });
        },

        filterData() {
            this.loadData();
        },

        refreshData() {
            this.loadData();
        },

        refreshSummary() {
            htmx.ajax('GET', '/summary', {
                target: '#summary-container',
                swap: 'innerHTML'
            });
        },

        exportToCSV() {
            const params = new URLSearchParams({
                prefix: this.activeTab,
                status: this.statusFilter,
                sort: this.sortBy,
                start_date: this.startDate,
                end_date: this.endDate
            });
            window.location.href = '/export-csv?' + params.toString();
        }
    }
}
window.appData = appData;
