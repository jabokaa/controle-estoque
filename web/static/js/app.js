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

// ---------- Chat do assistente de IA (pages/assistant.html) ----------
(() => {
    const form = document.getElementById("chat-form");
    const chatLog = document.getElementById("chat-log");
    if (!form || !chatLog) {
        return;
    }
    const textarea = document.getElementById("chat-message");
    const fileInput = document.getElementById("chat-attachments");
    const selectedFiles = document.getElementById("chat-selected-files");

    const scrollToBottom = () => {
        chatLog.scrollTop = chatLog.scrollHeight;
    };
    scrollToBottom();

    // Caixa de texto cresce com o conteúdo (até um limite, definido no CSS).
    const resizeTextarea = () => {
        textarea.style.height = "auto";
        textarea.style.height = textarea.scrollHeight + "px";
    };
    textarea.addEventListener("input", resizeTextarea);

    // Enter envia; Shift+Enter quebra a linha.
    textarea.addEventListener("keydown", (event) => {
        if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
            event.preventDefault();
            form.requestSubmit();
        }
    });

    // Mostra os arquivos escolhidos acima da caixa de texto.
    const showSelectedFiles = () => {
        const names = Array.from(fileInput.files).map((file) => file.name);
        selectedFiles.hidden = names.length === 0;
        selectedFiles.replaceChildren(...names.map((name) => {
            const chip = document.createElement("span");
            chip.className = "chat-file";
            chip.textContent = name;
            return chip;
        }));
    };
    fileInput.addEventListener("change", showSelectedFiles);

    // Sugestões da conversa vazia: preenchem a caixa de texto.
    document.addEventListener("click", (event) => {
        const suggestion = event.target.closest("[data-chat-suggestion]");
        if (suggestion) {
            textarea.value = suggestion.dataset.chatSuggestion;
            resizeTextarea();
            textarea.focus();
        }
    });

    // Antes de enviar: mostra a mensagem do usuário na hora (a resposta pode demorar).
    form.addEventListener("htmx:beforeRequest", () => {
        const text = textarea.value.trim();
        const files = Array.from(fileInput.files).map((file) => file.name);
        if (!text && files.length === 0) {
            return;
        }
        document.getElementById("chat-welcome")?.remove();

        const pending = document.createElement("div");
        pending.className = "chat-turn is-pending";
        const row = document.createElement("div");
        row.className = "chat-row is-user";
        const bubble = document.createElement("div");
        bubble.className = "chat-bubble is-user";
        if (text) {
            const paragraph = document.createElement("p");
            paragraph.className = "chat-text";
            paragraph.textContent = text;
            bubble.append(paragraph);
        }
        if (files.length > 0) {
            const fileList = document.createElement("div");
            fileList.className = "chat-files";
            files.forEach((name) => {
                const chip = document.createElement("span");
                chip.className = "chat-file";
                chip.textContent = name;
                fileList.append(chip);
            });
            bubble.append(fileList);
        }
        row.append(bubble);
        pending.append(row);
        chatLog.append(pending);
        scrollToBottom();
    });

    // Depois da resposta: tira a mensagem provisória (a resposta traz a definitiva) e limpa o formulário.
    form.addEventListener("htmx:afterRequest", (event) => {
        chatLog.querySelectorAll(".chat-turn.is-pending").forEach((pending) => pending.remove());
        if (event.detail.successful) {
            form.reset();
            showSelectedFiles();
            resizeTextarea();
        }
        textarea.focus();
    });

    chatLog.addEventListener("htmx:afterSwap", scrollToBottom);
    document.body.addEventListener("htmx:afterSettle", scrollToBottom);
})();
