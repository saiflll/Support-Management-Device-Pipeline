function appData() {
    console.log("forming.js initialized");
    return {
        activeTab: 'all',
        prefixes: [],
        searchQuery: '',
        statusFilter: 'all',
        sortBy: 'newest',
        startDate: '',
        endDate: '',

        init() {
            this.loadPrefixes();
            this.loadData();
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
                end_date: this.endDate,
                search: this.searchQuery
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

        async exportToCSV() {
            const table = document.querySelector('table');
            if (!table) return;

            let csv = [];
            const headers = Array.from(table.querySelectorAll('thead th')).map(th => th.textContent.trim());
            csv.push(headers.join(','));

            const rows = table.querySelectorAll('tbody tr');
            rows.forEach(row => {
                const cells = Array.from(row.querySelectorAll('td')).map(td => '"' + td.textContent.trim().replace(/"/g, '""') + '"');
                if (cells.length > 1) csv.push(cells.join(','));
            });

            const blob = new Blob([csv.join('\n')], {
                type: 'text/csv;charset=utf-8;'
            });
            const link = document.createElement('a');
            link.href = URL.createObjectURL(blob);
            link.download = `production_data_${new Date().toISOString().slice(0, 10)}.csv`;
            link.click();
        }
    }
}
window.appData = appData;
