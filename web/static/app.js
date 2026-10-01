document.addEventListener('DOMContentLoaded', () => {
    let currentArticles = [];
    let selectedIds = new Set();

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
    const selectAllCheckbox = document.getElementById('selectAllCheckbox');
    const selectionCounter = document.getElementById('selectionCounter');
    
    const loadingIndicator = document.getElementById('loadingIndicator');
    const errorMessage = document.getElementById('errorMessage');
    const errorText = document.getElementById('errorText');
    const resultsBody = document.getElementById('resultsBody');
    const colToggles = document.querySelectorAll('.col-toggle');

    const abstractModal = document.getElementById('abstractModal');
    const modalCloseBtn = document.getElementById('modalCloseBtn');
    const modalDismissBtn = document.getElementById('modalDismissBtn');
    const modalTitle = document.getElementById('modalTitle');
    const modalAuthors = document.getElementById('modalAuthors');
    const modalJournal = document.getElementById('modalJournal');
    const modalYear = document.getElementById('modalYear');
    const modalCitations = document.getElementById('modalCitations');
    const modalDoi = document.getElementById('modalDoi');
    const modalSourceBadge = document.getElementById('modalSourceBadge');
    const modalAbstractContent = document.getElementById('modalAbstractContent');
    const modalExternalLink = document.getElementById('modalExternalLink');

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

    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', (e) => {
            const checkAll = e.target.checked;
            if (checkAll) {
                currentArticles.forEach(a => selectedIds.add(a._id));
            } else {
                selectedIds.clear();
            }
            const cbs = resultsBody.querySelectorAll('.article-checkbox');
            cbs.forEach(cb => {
                cb.checked = checkAll;
                const row = cb.closest('tr');
                if (row) {
                    row.classList.toggle('row-selected', checkAll);
                }
            });
            updateSelectionUI();
        });
    }

    function updateSelectionUI() {
        const total = currentArticles.length;
        const selectedCount = currentArticles.filter(a => selectedIds.has(a._id)).length;

        if (total === 0) {
            if (selectAllCheckbox) {
                selectAllCheckbox.checked = false;
                selectAllCheckbox.indeterminate = false;
                selectAllCheckbox.disabled = true;
            }
            if (selectionCounter) {
                selectionCounter.classList.add('hidden');
                selectionCounter.classList.remove('has-selection');
                selectionCounter.textContent = '0 de 0 artigos selecionados';
            }
            if (exportCsvBtn) exportCsvBtn.disabled = true;
            if (exportXlsxBtn) exportXlsxBtn.disabled = true;
            return;
        }

        if (selectAllCheckbox) selectAllCheckbox.disabled = false;
        if (selectionCounter) {
            selectionCounter.classList.remove('hidden');
            selectionCounter.textContent = `${selectedCount} de ${total} artigos selecionados`;
        }

        if (selectedCount === 0) {
            if (selectAllCheckbox) {
                selectAllCheckbox.checked = false;
                selectAllCheckbox.indeterminate = false;
            }
            if (selectionCounter) selectionCounter.classList.remove('has-selection');
            if (exportCsvBtn) exportCsvBtn.disabled = true;
            if (exportXlsxBtn) exportXlsxBtn.disabled = true;
        } else if (selectedCount === total) {
            if (selectAllCheckbox) {
                selectAllCheckbox.checked = true;
                selectAllCheckbox.indeterminate = false;
            }
            if (selectionCounter) selectionCounter.classList.add('has-selection');
            if (exportCsvBtn) exportCsvBtn.disabled = false;
            if (exportXlsxBtn) exportXlsxBtn.disabled = false;
        } else {
            if (selectAllCheckbox) {
                selectAllCheckbox.checked = false;
                selectAllCheckbox.indeterminate = true;
            }
            if (selectionCounter) selectionCounter.classList.add('has-selection');
            if (exportCsvBtn) exportCsvBtn.disabled = false;
            if (exportXlsxBtn) exportXlsxBtn.disabled = false;
        }
    }

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

        const providerCheckboxes = document.querySelectorAll('.provider-toggle');
        const selectedProviders = [];
        providerCheckboxes.forEach(cb => {
            if (cb.checked) {
                selectedProviders.push(cb.value);
            }
        });

        if (selectedProviders.length === 0) {
            showError('Selecione pelo menos uma fonte de busca (OpenAlex ou CrossRef).');
            return;
        }

        const loadingSubtext = document.querySelector('.loading-subtext');
        if (loadingSubtext) {
            const providerNames = selectedProviders.map(p => p.toLowerCase() === 'openalex' ? 'OpenAlex' : (p.toLowerCase() === 'crossref' ? 'CrossRef' : p));
            loadingSubtext.textContent = `Buscando em tempo real: ${providerNames.join(' e ')}`;
        }

        loadingIndicator.classList.remove('hidden');
        errorMessage.classList.add('hidden');
        resultsBody.innerHTML = '';
        searchButton.disabled = true;
        selectedIds.clear();
        updateSelectionUI();

        try {
            const searchPayload = {
                query: query,
                limit: limit,
                min_year: minYear,
                min_citations: minCitations,
                providers: selectedProviders
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

            filteredArticles.forEach((a, idx) => {
                if (!a._id) {
                    a._id = a.id || a.doi || ('art_' + idx + '_' + Date.now());
                }
            });

            currentArticles = filteredArticles;
            currentArticles.forEach(a => selectedIds.add(a._id));
            renderTable(filteredArticles);

        } catch (error) {
            showError(error.message);
            currentArticles = [];
            selectedIds.clear();
            updateSelectionUI();
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
                    <td colspan="9" class="empty-state">
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
            updateSelectionUI();
            return;
        }

        articles.forEach(article => {
            const tr = document.createElement('tr');
            const isSelected = selectedIds.has(article._id);
            if (isSelected) {
                tr.classList.add('row-selected');
            }

            const tdSelect = document.createElement('td');
            tdSelect.className = 'col-select text-center';
            const cb = document.createElement('input');
            cb.type = 'checkbox';
            cb.className = 'article-checkbox';
            cb.dataset.id = article._id;
            cb.checked = isSelected;
            cb.setAttribute('aria-label', `Selecionar artigo ${article.title || ''}`);
            tdSelect.appendChild(cb);
            tr.appendChild(tdSelect);

            cb.addEventListener('change', (e) => {
                e.stopPropagation();
                if (cb.checked) {
                    selectedIds.add(article._id);
                    tr.classList.add('row-selected');
                } else {
                    selectedIds.delete(article._id);
                    tr.classList.remove('row-selected');
                }
                updateSelectionUI();
            });

            tdSelect.addEventListener('click', (e) => {
                if (e.target !== cb) {
                    cb.checked = !cb.checked;
                    cb.dispatchEvent(new Event('change'));
                }
            });
            
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

            const colsHtml = `
                <td class="col-title">${titleContent}</td>
                <td class="col-authors">${authorsStr}</td>
                <td class="col-year text-center">${year}</td>
                <td class="col-journal">${journalStr}</td>
                <td class="col-citations text-right">${citations.toLocaleString('pt-BR')}</td>
                <td class="col-doi">${doiContent}</td>
                <td class="col-source text-center"><span class="source-badge source-${providerClass}">${providerName}</span></td>
                <td class="col-actions text-center">
                    <button class="btn-view-abstract" type="button" title="Ver Resumo" aria-label="Ver Resumo">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                            <polyline points="14 2 14 8 20 8"></polyline>
                            <line x1="16" y1="13" x2="8" y2="13"></line>
                            <line x1="16" y1="17" x2="8" y2="17"></line>
                            <polyline points="10 9 9 9 8 9"></polyline>
                        </svg>
                        <span>Resumo</span>
                    </button>
                </td>
            `;
            tr.insertAdjacentHTML('beforeend', colsHtml);

            const btnAbstract = tr.querySelector('.btn-view-abstract');
            if (btnAbstract) {
                btnAbstract.addEventListener('click', (e) => {
                    e.stopPropagation();
                    openAbstractModal(article);
                });
            }

            resultsBody.appendChild(tr);
        });

        colToggles.forEach(checkbox => {
            if (!checkbox.checked) {
                const colClass = checkbox.dataset.col;
                resultsBody.querySelectorAll(`.${colClass}`).forEach(el => el.classList.add('col-hidden'));
            }
        });

        updateSelectionUI();
    }

    function showError(msg) {
        errorText.textContent = msg;
        errorMessage.classList.remove('hidden');
    }

    function escapeHtml(str) {
        if (!str) return '';
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    function openAbstractModal(article) {
        if (!abstractModal) return;

        modalTitle.textContent = article.title || 'Sem título';
        modalAuthors.textContent = article.authors && article.authors.length ? article.authors.join(', ') : 'Não informado';
        modalJournal.textContent = article.journal || 'Não informado';
        modalYear.textContent = article.publication_year || article.publicationYear || article.year || 'Não informado';

        const citations = article.citation_count !== undefined && article.citation_count !== null ? article.citation_count : (article.citations || 0);
        modalCitations.textContent = citations.toLocaleString('pt-BR');

        if (article.doi) {
            const doiLink = article.doi.startsWith('http') ? article.doi : `https://doi.org/${article.doi}`;
            modalDoi.innerHTML = `<a href="${doiLink}" target="_blank" rel="noopener noreferrer">${escapeHtml(article.doi)}</a>`;
        } else {
            modalDoi.textContent = 'Não informado';
        }

        const providerName = article.source_provider || 'Desconhecido';
        const providerClass = providerName.toLowerCase().replace(/[^a-z0-9]/g, '');
        modalSourceBadge.innerHTML = `<span class="source-badge source-${providerClass}">${escapeHtml(providerName)}</span>`;

        const rawAbstract = (article.abstract || '').trim();
        if (rawAbstract) {
            modalAbstractContent.innerHTML = `<p class="modal-abstract-text">${escapeHtml(rawAbstract)}</p>`;
        } else {
            modalAbstractContent.innerHTML = `<p class="modal-abstract-empty">Resumo não disponibilizado publicamente pelo provedor indexador.</p>`;
        }

        const targetUrl = article.url || (article.doi ? (article.doi.startsWith('http') ? article.doi : `https://doi.org/${article.doi}`) : '');
        if (targetUrl) {
            modalExternalLink.href = targetUrl;
            modalExternalLink.style.display = 'inline-flex';
        } else {
            modalExternalLink.style.display = 'none';
        }

        abstractModal.classList.remove('hidden');
        document.body.classList.add('modal-open');
        if (modalCloseBtn) modalCloseBtn.focus();
    }

    function closeAbstractModal() {
        if (!abstractModal) return;
        abstractModal.classList.add('hidden');
        document.body.classList.remove('modal-open');
    }

    if (modalCloseBtn) {
        modalCloseBtn.addEventListener('click', closeAbstractModal);
    }

    if (modalDismissBtn) {
        modalDismissBtn.addEventListener('click', closeAbstractModal);
    }

    if (abstractModal) {
        abstractModal.addEventListener('click', (e) => {
            if (e.target === abstractModal) {
                closeAbstractModal();
            }
        });
    }

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && abstractModal && !abstractModal.classList.contains('hidden')) {
            closeAbstractModal();
        }
    });

    function exportToXmlSpreadsheet(articles) {
        const escapeXml = (str) => {
            if (str === null || str === undefined) return '';
            return String(str)
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&apos;');
        };

        let rowsXml = '';
        const headers = ['ID', 'Título', 'Autores', 'Ano', 'DOI', 'URL', 'Periódico', 'Citações', 'Resumo', 'Provedor'];
        rowsXml += '<Row>';
        headers.forEach(h => {
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(h)}</Data></Cell>`;
        });
        rowsXml += '</Row>';

        articles.forEach(a => {
            const authors = a.authors && a.authors.length ? a.authors.join(', ') : '';
            const year = a.year || a.publication_year || a.publicationYear || '';
            const citations = a.citations !== undefined ? a.citations : (a.citation_count || 0);
            rowsXml += '<Row>';
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.id || '')}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.title || '')}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(authors)}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="Number">${escapeXml(year)}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.doi || '')}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.url || '')}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.journal || '')}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="Number">${escapeXml(citations)}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.abstract || '')}</Data></Cell>`;
            rowsXml += `<Cell><Data ss:Type="String">${escapeXml(a.source_provider || '')}</Data></Cell>`;
            rowsXml += '</Row>';
        });

        const xmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:o="urn:schemas-microsoft-com:office:office"
 xmlns:x="urn:schemas-microsoft-com:office:excel"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:html="http://www.w3.org/TR/REC-html40">
 <Worksheet ss:Name="Artigos">
  <Table>
   ${rowsXml}
  </Table>
 </Worksheet>
</Workbook>`;

        const blob = new Blob([xmlTemplate], { type: 'application/vnd.ms-excel;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'noosfera_artigos.xlsx';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    if (exportCsvBtn) exportCsvBtn.addEventListener('click', () => exportData('csv'));
    if (exportXlsxBtn) exportXlsxBtn.addEventListener('click', () => exportData('xlsx'));

    async function exportData(format) {
        const selectedArticles = currentArticles.filter(a => selectedIds.has(a._id));
        if (!selectedArticles || selectedArticles.length === 0) {
            alert('Nenhum artigo selecionado para exportação. Selecione pelo menos um artigo.');
            return;
        }

        try {
            const response = await fetch('/api/v1/export', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    articles: selectedArticles,
                    format: format
                })
            });

            if (!response.ok) {
                if (format === 'xlsx') {
                    exportToXmlSpreadsheet(selectedArticles);
                    return;
                }
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
            if (format === 'xlsx') {
                exportToXmlSpreadsheet(selectedArticles);
            } else {
                alert(`Falha ao exportar: ${error.message}`);
            }
        }
    }

    updateSelectionUI();
});
