document.addEventListener('DOMContentLoaded', function() {
    // Элементы интерфейса
    const createButton = document.getElementById('createButton');
    const createModal = document.getElementById('createModal');
    const closeModal = document.querySelector('.close');
    const cancelButton = document.getElementById('cancelCreate');
    const submitButton = document.getElementById('submitVoting');
    const addOptionButton = document.getElementById('addOption');
    const optionsContainer = document.getElementById('optionsContainer');

    // Открытие модального окна
    createButton.addEventListener('click', () => {
        createModal.style.display = 'flex';
    });

    // Закрытие модального окна
    const closeModalHandler = () => {
        createModal.style.display = 'none';
    };

    closeModal.addEventListener('click', closeModalHandler);
    cancelButton.addEventListener('click', closeModalHandler);

    // Добавление нового варианта ответа
    addOptionButton.addEventListener('click', () => {
        if (optionsContainer.children.length >= 4) {
            alert('Максимум 4 варианта ответа');
            return;
        }

        const newOption = document.createElement('input');
        newOption.type = 'text';
        newOption.className = 'vote-option';
        newOption.placeholder = `Вариант ${optionsContainer.children.length + 1}`;
        newOption.maxLength = 100;
        optionsContainer.appendChild(newOption);
    });

    // Отправка формы
    submitButton.addEventListener('click', async () => {
        const votingData = {
            title: document.getElementById('voteTitle').value,
            description: document.getElementById('voteDescription').value,
            is_private: document.querySelector('input[name="voteType"]:checked').value === 'private',
            min_votes: parseInt(document.getElementById('minVotes').value),
            end_date: new Date(document.getElementById('endDate').value).toISOString(),
            options: Array.from(document.querySelectorAll('.vote-option'))
                .map(input => input.value)
                .filter(text => text.trim() !== '')
        };

        if (!validateVoting(votingData)) return;

        try {
            const response = await fetch('/voting', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(votingData)
            });

            if (response.ok) {
                const result = await response.json();

                // Кешируем данные в localStorage
                //localStorage.setItem(`voting_${result.voting_id}`, JSON.stringify(result));

                alert(`Голосование создано! ID: ${result.voting_id}`);
                closeModalHandler();
                loadVotings();
            }
            else {
                const errorText = await response.text();
                console.error('Ошибка от сервера:', errorText);
                alert('Ошибка при создании голосования: ' + errorText);
            }

        } catch (error) {
            console.error('Error:', error);
            alert('Ошибка при создании голосования');
        }
    });

    // Загрузка списка голосований
    async function loadVotings() {
        try {
            const response = await fetch('/voting'); // Запрос без параметра 'type=all'
            if (response.ok) {
                const votings = await response.json();
                renderVotings(votings);
            } else {
                console.error('Failed to load votings:', response.status, response.statusText);
            }
        } catch (error) {
            console.error('Error loading votings:', error);
        }
    }

    // Валидация данных
    function validateVoting(data) {
        if (!data.title.trim()) {
            alert('Введите название голосования');
            return false;
        }

        if (data.options.length < 2) {
            alert('Добавьте хотя бы 2 варианта ответа');
            return false;
        }

        if (!data.end_date) {
            alert('Укажите дату окончания');
            return false;
        }

        return true;
    }

    // Первоначальная загрузка
    loadVotings();
});

// Рендер списка голосований
function renderVotings(votings) {
    const container = document.getElementById('votingsList');
    container.innerHTML = '';

    votings.forEach(voting => {
        // Сохраняем в кеш (если вдруг пришло с сервера)
        localStorage.setItem(`voting_${voting.id}`, JSON.stringify(voting));

        const card = document.createElement('div');
        card.className = 'voting-card';
        card.innerHTML = `
            <h3 class="voting-title">${voting.title}</h3>
            <p class="voting-description">${voting.description || 'Нет описания'}</p>
            <div class="voting-meta">
                <span>До ${new Date(voting.end_date).toLocaleString()}</span>
                <span>${voting.is_private ? '🔒 Приватное' : '🌍 Публичное'}</span>
            </div>
        `;
        container.appendChild(card);
    });
}
