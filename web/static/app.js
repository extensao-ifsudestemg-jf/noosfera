document.addEventListener('DOMContentLoaded', () => {
    let currentArticles = [];

    const searchInput = document.getElementById('searchInput');
    const limitInput = document.getElementById('limitInput');
    const yearStartInput = document.getElementById('yearStart');
    const yearEndInput = document.getElementById('yearEnd');
    const minCitationsInput = document.getElementById('minCitations');
    const journalKeywordInput = document.getElementById('journalKeyword');
    const crossrefEmailInput = document.getElementById('crossref-email');
    
    const searchButton = document.getElementById('searchButton');
    const toggleFiltersBtn = document.getElementById('toggleFiltersBtn');
    const filtersPanel = document.getElementById('filtersPanel');
    const themeToggleBtn = document.getElementById('themeToggleBtn');
    
    const exportCsvBtn = document.getElementById('exportCsvBtn');
    const exportXlsxBtn = document.getElementById('exportXlsxBtn');
    
    const loadingIndicator = document.getElementById('loadingIndicator');
    const errorMessage = document.getElementById('errorMessage');
    const errorText = document.getElementById('errorText');
    const resultsBody = document.getElementById('resultsBody');
    const colToggles = document.querySelectorAll('.col-toggle');

    function initTheme() {
        const savedTheme = localStorage.getItem('theme');
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)');
        
        let initialTheme = savedTheme || (systemPrefersDark.matches ? 'dark' : 'light');
        document.documentElement.setAttribute('data-theme', initialTheme);

        if (themeToggleBtn) {
            themeToggleBtn.addEventListener('click', () => {
                const currentTheme = document.documentElement.getAttribute('data-theme');
                const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
                
                document.documentElement.setAttribute('data-theme', newTheme);
                localStorage.setItem('theme', newTheme);
            });
        }

        systemPrefersDark.addEventListener('change', (e) => {
            if (!localStorage.getItem('theme')) {
                const newTheme = e.matches ? 'dark' : 'light';
                document.documentElement.setAttribute('data-theme', newTheme);
            }
        });
    }

    initTheme();

    if (crossrefEmailInput) {
        const savedEmail = localStorage.getItem('crossref_email');
        if (savedEmail) {
            crossrefEmailInput.value = savedEmail;
        }

        crossrefEmailInput.addEventListener('change', (e) => {
            const val = e.target.value.trim();
            if (val) {
                localStorage.setItem('crossref_email', val);
            } else {
                localStorage.removeItem('crossref_email');
            }
        });
    }

    if (toggleFiltersBtn && filtersPanel) {
        toggleFiltersBtn.addEventListener('click', (e) => {
            const isCollapsed = filtersPanel.classList.toggle('collapsed');
            const isExpanded = !isCollapsed;
            e.currentTarget.classList.toggle('active', isExpanded);
            e.currentTarget.setAttribute('aria-expanded', isExpanded.toString());
        });
    }

    colToggles.forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
            const colClass = e.target.dataset.col;
            const elements = document.querySelectorAll(`.${colClass}`);
            elements.forEach(el => {
                if (e.target.checked) {
                    el.classList.remove('col-hidden');
                } else {
                    el.classList.add('col-hidden');
                }
            });
        });
    });

    if (searchButton) {
        searchButton.addEventListener('click', performSearch);
    }
    
    if (searchInput) {
        searchInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                performSearch();
            }
        });
    }

    async function performSearch() {
        const query = searchInput.value.trim();
        if (query.length < 3) {
            showError('O termo de busca deve ter no mínimo 3 caracteres.');
            return;
        }

        const limit = parseInt(limitInput.value) || 20;
        const minYear = parseInt(yearStartInput.value) || 0;
        const minCitations = parseInt(minCitationsInput.value) || 0;
        
        const maxYearStr = yearEndInput.value;
        const maxYear = maxYearStr ? parseInt(maxYearStr) : 0;
        const journalKeyword = journalKeywordInput.value.trim().toLowerCase();

        const userEmail = crossrefEmailInput ? crossrefEmailInput.value.trim() : '';
        if (userEmail) {
            localStorage.setItem('crossref_email', userEmail);
        } else {
            localStorage.removeItem('crossref_email');
        }

        loadingIndicator.classList.remove('hidden');
        errorMessage.classList.add('hidden');
        resultsBody.innerHTML = '';
        searchButton.disabled = true;

        try {
            const searchPayload = {
                query: query,
                limit: limit,
                min_year: minYear,
                min_citations: minCitations
            };

            if (userEmail) {
                searchPayload.user_email = userEmail;
            }

            const response = await fetch('/api/v1/search', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(searchPayload)
            });

            if (!response.ok) {
                let errData;
                try {
                    errData = await response.json();
                } catch (e) {
                    errData = { detail: `Erro HTTP: ${response.status}` };
                }
                throw new Error(errData.detail || 'Erro ao consultar a API');
            }

            const data = await response.json();
            let filteredArticles = data.articles || [];

            if (maxYear > 0) {
                filteredArticles = filteredArticles.filter(a => {
                    const yr = a.publication_year || a.publicationYear || a.year || 0;
                    return yr <= maxYear && yr > 0;
                });
            }
            if (journalKeyword) {
                filteredArticles = filteredArticles.filter(a => 
                    a.journal && a.journal.toLowerCase().includes(journalKeyword)
                );
            }

            currentArticles = filteredArticles;
            renderTable(filteredArticles);

        } catch (error) {
            showError(error.message);
            currentArticles = [];
        } finally {
            loadingIndicator.classList.add('hidden');
            searchButton.disabled = false;
        }
    }

    function renderTable(articles) {
        resultsBody.innerHTML = '';

        if (articles.length === 0) {
            resultsBody.innerHTML = `
                <tr>
                    <td colspan="7" class="empty-state">
                        <div class="empty-state-icon">
                            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
                                <circle cx="11" cy="11" r="8"></circle>
                                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                            </svg>
                        </div>
                        <p class="empty-title">Nenhum artigo encontrado</p>
                        <p class="empty-subtitle">Tente ajustar os termos de busca ou remover alguns filtros aplicados.</p>
                    </td>
                </tr>`;
            return;
        }

        articles.forEach(article => {
            const tr = document.createElement('tr');
            
            const titleContent = article.url 
                ? `<a href="${article.url}" target="_blank" rel="noopener noreferrer" title="Acessar publicação">${article.title}</a>`
                : article.title;
                
            let doiContent = '-';
            if (article.doi) {
                const doiLink = article.doi.startsWith('http') ? article.doi : `https://doi.org/${article.doi}`;
                doiContent = `<a href="${doiLink}" target="_blank" rel="noopener noreferrer" title="Ver DOI">${article.doi}</a>`;
            }

            const authorsStr = article.authors && article.authors.length ? article.authors.join(', ') : '-';
            const journalStr = article.journal || '-';

            const year = article.publication_year || article.publicationYear || '-';
            const citations = article.citation_count !== undefined && article.citation_count !== null ? article.citation_count : (article.citations || 0);

            const providerName = article.source_provider || 'Desconhecido';
            const providerClass = providerName.toLowerCase().replace(/[^a-z0-9]/g, '');

            tr.innerHTML = `
                <td class="col-title">${titleContent}</td>
                <td class="col-authors">${authorsStr}</td>
                <td class="col-year text-center">${year}</td>
                <td class="col-journal">${journalStr}</td>
                <td class="col-citations text-right">${citations.toLocaleString('pt-BR')}</td>
                <td class="col-doi">${doiContent}</td>
                <td class="col-source text-center"><span class="source-badge source-${providerClass}">${providerName}</span></td>
            `;
            resultsBody.appendChild(tr);
        });

        colToggles.forEach(checkbox => {
            if (!checkbox.checked) {
                const colClass = checkbox.dataset.col;
                resultsBody.querySelectorAll(`.${colClass}`).forEach(el => el.classList.add('col-hidden'));
            }
        });
    }

    function showError(msg) {
        errorText.textContent = msg;
        errorMessage.classList.remove('hidden');
    }

    if (exportCsvBtn) exportCsvBtn.addEventListener('click', () => exportData('csv'));
    if (exportXlsxBtn) exportXlsxBtn.addEventListener('click', () => exportData('xlsx'));

    async function exportData(format) {
        if (!currentArticles || currentArticles.length === 0) {
            alert('Realize uma busca primeiro e garanta que há artigos para exportar.');
            return;
        }

        try {
            const response = await fetch('/api/v1/export', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    articles: currentArticles,
                    format: format
                })
            });

            if (!response.ok) {
                const errData = await response.json().catch(() => null);
                const errorMsg = errData && errData.detail ? errData.detail : `Erro na exportação (Status: ${response.status})`;
                throw new Error(errorMsg);
            }

            const blob = await response.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `noosfera_artigos.${format}`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            
        } catch (error) {
            alert(`Falha ao exportar: ${error.message}`);
        }
    }
});
