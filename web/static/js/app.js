// Comportamentos de tela compartilhados por todas as páginas.

// Modal de confirmação de exclusão (components/confirm_delete_modal):
// quando o modal abre, preenche o formulário com os dados do botão que foi clicado.
// O listener fica no document para funcionar também com as linhas trocadas pelo HTMX.
document.addEventListener("show.bs.modal", (event) => {
    const modal = event.target;
    const trigger = event.relatedTarget;
    if (!modal.matches("[data-confirm-delete]") || !trigger) {
        return;
    }

    modal.querySelector("form").action = trigger.dataset.deleteUrl;
    modal.querySelector("[data-delete-name]").textContent = trigger.dataset.deleteName;
    // Depois de excluir, volta para a mesma listagem (mesmo filtro, ordem e página).
    modal.querySelector("input[name=return_to]").value = window.location.pathname + window.location.search;
});
