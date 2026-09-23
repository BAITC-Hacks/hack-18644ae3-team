/* Shared UI translations. Stored values, identifiers and user input stay unchanged. */
const CareerQuestI18n = (() => {
  const rows = `
Language|Язык|Тіл
Sign in|Войти|Кіру
Signing in…|Вход…|Кіру…
Sign in failed|Не удалось войти|Кіру мүмкін болмады
Sign in to your workspace|Вход в личный кабинет|Жеке кабинетке кіру
Welcome back|С возвращением|Қайта қош келдіңіз
Email address|Электронная почта|Электрондық пошта
Email|Электронная почта|Электрондық пошта
Work email|Рабочая почта|Жұмыс поштасы
Password|Пароль|Құпиясөз
Enter password|Введите пароль|Құпиясөзді енгізіңіз
Full name|Полное имя|Толық аты-жөні
Employee ID|ID сотрудника|Қызметкер ID
(optional; HR will verify)|(необязательно; HR проверит)|(міндетті емес; HR тексереді)
Leave empty if you don't have one|Оставьте пустым, если ID нет|ID болмаса, бос қалдырыңыз
Need an account? Request access|Нет аккаунта? Запросить доступ|Аккаунтыңыз жоқ па? Қолжетімділік сұрау
Submit for HR approval|Отправить на одобрение HR|HR мақұлдауына жіберу
Back to sign in|Вернуться ко входу|Кіруге оралу
Demo accounts|Демоаккаунты|Демоаккаунттар
All demo accounts use password|Пароль всех демоаккаунтов|Барлық демоаккаунттардың құпиясөзі
Registration failed|Не удалось зарегистрироваться|Тіркелу мүмкін болмады
Registration submitted for HR approval.|Заявка отправлена на одобрение HR.|Өтінім HR мақұлдауына жіберілді.
invalid email or password|Неверная почта или пароль|Пошта немесе құпиясөз қате
Career development, connected|Всё для карьерного развития|Мансаптық дамудың біртұтас жүйесі
Every activity should move someone forward.|Каждое обучение — шаг вперёд.|Әр оқу — алға жасалған қадам.
One platform for personal career journeys, organization-wide development, and learning activity management.|Единая платформа для карьеры сотрудников, развития компании и управления обучением.|Қызметкерлердің мансабы, компанияның дамуы және оқуды басқаруға арналған бірыңғай платформа.
Understand gaps and take the right next step.|Узнайте, какие навыки развивать и что делать дальше.|Қай дағдыларды дамыту және келесі қадамды анықтау.
Guide development across the organization.|Управляйте развитием сотрудников компании.|Компания қызметкерлерінің дамуын басқарыңыз.
Build activities that close real skill gaps.|Создавайте обучение для развития нужных навыков.|Қажетті дағдыларды дамытатын оқу бағдарламаларын жасаңыз.
Your role determines what information and tools you can access.|Ваша роль определяет доступные данные и инструменты.|Сіздің рөліңіз қолжетімді ақпарат пен құралдарды анықтайды.
Secure demo workspace|Защищённый демокабинет|Қорғалған демокабинет
HR teams|HR-команды|HR топтары
L&D teams|Команды обучения и развития|Оқыту және дамыту топтары
HR workspace|Кабинет HR|HR кабинеті
Employee workspace|Кабинет сотрудника|Қызметкер кабинеті
L&D workspace|Кабинет обучения и развития|Оқыту және дамыту кабинеті
All employees and analytics|Все сотрудники и аналитика|Барлық қызметкерлер мен талдау
Personal career journey|Личный карьерный путь|Жеке мансап жолы
Courses and events|Курсы и мероприятия|Курстар мен іс-шаралар
My workspace|Мой кабинет|Менің кабинетім
My Career|Моя карьера|Менің мансабым
My Quests|Мои задания|Менің тапсырмаларым
Skills|Навыки|Дағдылар
Career Path|Карьерный путь|Мансап жолы
AI Navigator|ИИ-навигатор|ЖИ навигаторы
AI Career Navigator|Карьерный ИИ-навигатор|Мансаптық ЖИ навигаторы
Career Navigator|Карьерный навигатор|Мансап навигаторы
Profile|Профиль|Профиль
My Profile|Мой профиль|Менің профилім
Log out|Выйти|Шығу
Personal career space|Личный карьерный кабинет|Жеке мансап кабинеті
Loading…|Загрузка…|Жүктелуде…
Loading your career journey…|Загрузка карьерного пути…|Мансап жолы жүктелуде…
Loading secure workspace…|Загрузка кабинета…|Кабинет жүктелуде…
Loading learning catalog…|Загрузка каталога обучения…|Оқу каталогы жүктелуде…
Your career dashboard|Ваша карьера|Сіздің мансабыңыз
See exactly where you are, what is missing, and the best next step.|Ваш прогресс, недостающие навыки и следующий шаг.|Сіздің ілгерілеуіңіз, жетіспейтін дағдылар мен келесі қадам.
Edit career goal|Изменить карьерную цель|Мансаптық мақсатты өзгерту
Best next step|Следующий шаг|Келесі қадам
Recommended next quest|Рекомендуемое задание|Ұсынылған тапсырма
All my quests|Все мои задания|Барлық тапсырмаларым
Promotion blockers|Что мешает повышению|Жоғарылауға кедергілер
Critical skill gaps|Критические пробелы в навыках|Маңызды дағды жетіспеушіліктері
All skills|Все навыки|Барлық дағдылар
In motion|В процессе|Орындалуда
Current activities|Текущие активности|Ағымдағы іс-шаралар
Development activities|Развивающие активности|Дамыту іс-шаралары
Everything recommended, underway, completed, or required for you.|Рекомендации, текущие, завершённые и обязательные активности.|Ұсынылған, орындалып жатқан, аяқталған және міндетті іс-шаралар.
Search the course and event catalog|Поиск по каталогу курсов и мероприятий|Курстар мен іс-шаралар каталогынан іздеу
Recommended|Рекомендовано|Ұсынылған
Planned / Enrolled|Запланировано / Записан|Жоспарланған / Тіркелген
Planned / In progress|Запланировано / В процессе|Жоспарланған / Орындалуда
In Progress|В процессе|Орындалуда
Completed|Завершено|Аяқталған
Mandatory|Обязательно|Міндетті
Optional|По желанию|Қосымша
Your capability map|Карта ваших навыков|Сіздің дағдылар картаңыз
Critical gaps|Критические пробелы|Маңызды жетіспеушіліктер
Critical blockers|Критические препятствия|Негізгі кедергілер
Must reach the target level.|Необходимо достичь целевого уровня.|Мақсатты деңгейге жету қажет.
Growth skills|Навыки для роста|Өсуге арналған дағдылар
Increase overall readiness.|Повышайте готовность к следующей роли.|Келесі лауазымға дайындығыңызды арттырыңыз.
Visible progress|Видимый прогресс|Көрінетін ілгерілеу
How each activity builds skills and moves you toward your goal.|Как обучение развивает навыки и приближает к цели.|Оқу дағдыларды қалай дамытып, мақсатқа жақындататынын көріңіз.
Change goal|Изменить цель|Мақсатты өзгерту
Your career guide|Ваш карьерный помощник|Сіздің мансап көмекшіңіз
Recommendations are calculated by the engine; Navigator explains them.|Система подбирает активности, навигатор объясняет рекомендации.|Жүйе іс-шараларды таңдайды, навигатор ұсыныстарды түсіндіреді.
Grounded guidance|Рекомендации на основе данных|Деректерге негізделген кеңестер
Uses only your career data|На основе ваших карьерных данных|Мансаптық деректеріңізге негізделген
What should I do next?|Что мне делать дальше?|Әрі қарай не істеуім керек?
What blocks my promotion?|Что мешает моему повышению?|Жоғарылауыма не кедергі?
Personal details|Личные данные|Жеке деректер
Your role, team, career goal, and assessed capability data.|Ваша роль, команда, карьерная цель и оценка навыков.|Сіздің лауазымыңыз, тобыңыз, мансаптық мақсатыңыз және дағды бағаларыңыз.
Assessed skills|Оценённые навыки|Бағаланған дағдылар
Effective levels include completed activities since your last review.|Уровни учитывают обучение после последней оценки.|Деңгейлер соңғы бағалаудан кейінгі аяқталған оқуды ескереді.
My main quest|Моя главная цель|Менің басты мақсатым
Choose a career goal|Выберите карьерную цель|Мансаптық мақсатты таңдаңыз
Your path and recommendations will adapt immediately.|Путь и рекомендации обновятся сразу.|Жолыңыз бен ұсыныстар бірден жаңартылады.
Target role|Целевая роль|Мақсатты лауазым
Target grade|Целевой уровень|Мақсатты деңгей
Target roles|Целевые роли|Мақсатты лауазымдар
Target grades|Целевые уровни|Мақсатты деңгейлер
Clear goal|Убрать цель|Мақсатты алып тастау
Save goal|Сохранить цель|Мақсатты сақтау
Cancel|Отмена|Бас тарту
Dashboard|Обзор|Шолу
Employees|Сотрудники|Қызметкерлер
Employee|Сотрудник|Қызметкер
Events|Мероприятия|Іс-шаралар
Analytics|Аналитика|Талдау
Registration requests|Заявки на регистрацию|Тіркелу өтінімдері
HR secure view|Кабинет HR|HR кабинеті
Snapshot|Срез данных|Деректер күні
Employee profile|Профиль сотрудника|Қызметкер профилі
Refresh|Обновить|Жаңарту
Authorized access|Авторизованный доступ|Рұқсат етілген қолжетімділік
Employee development|Развитие сотрудников|Қызметкерлерді дамыту
Employee overview|Обзор сотрудника|Қызметкер туралы шолу
Review career progress and recommended development.|Карьерный прогресс и рекомендации по развитию.|Мансаптық ілгерілеу және даму ұсыныстары.
Edit employee goal|Изменить цель сотрудника|Қызметкердің мақсатын өзгерту
Recommended development|Рекомендации по развитию|Даму бойынша ұсыныстар
Next best activities|Рекомендуемые активности|Ұсынылатын іс-шаралар
View all|Посмотреть все|Барлығын көру
Focus area|В фокусе|Назарда
Skill gaps|Пробелы в навыках|Дағды жетіспеушіліктері
Full skill view|Все навыки|Барлық дағдылар
Activity|Активность|Іс-шара
Latest development|Последние активности|Соңғы іс-шаралар
Development progress|Прогресс развития|Даму барысы
Activity status|Статус активностей|Іс-шаралар күйі
Organization|Организация|Ұйым
Search the organization and open an employee development profile.|Найдите сотрудника и откройте его профиль развития.|Қызметкерді тауып, оның даму профилін ашыңыз.
Add employee|Добавить сотрудника|Қызметкер қосу
Search name, email, ID, department, team, role or grade|Поиск по имени, почте, ID, отделу, команде, роли или уровню|Аты, поштасы, ID, бөлімі, тобы, лауазымы немесе деңгейі бойынша іздеу
Department|Отдел|Бөлім
Team|Команда|Топ
Manager|Руководитель|Басшы
Location|Местоположение|Орналасқан жері
Phone|Телефон|Телефон
All roles|Все роли|Барлық лауазымдар
All grades|Все уровни|Барлық деңгейлер
Role|Роль|Лауазым
Grade|Уровень|Деңгей
Career goal|Карьерная цель|Мансаптық мақсат
Account access|Доступ к аккаунту|Аккаунтқа қолжетімділік
Create a new employee with an automatic ID, or link an existing employee.|Создайте сотрудника с автоматическим ID или свяжите с существующим.|Автоматты ID бар қызметкер жасаңыз немесе бұрынғы қызметкерге байланыстырыңыз.
Applicant|Заявитель|Өтініш беруші
Email / requested ID|Почта / запрошенный ID|Пошта / сұралған ID
Approval setup|Настройки одобрения|Мақұлдау параметрлері
Status|Статус|Күйі
Approve|Одобрить|Мақұлдау
Reject|Отклонить|Қабылдамау
Pending|Ожидает одобрения|Мақұлдауды күтуде
Active|Активен|Белсенді
Rejected|Отклонено|Қабылданбаған
Create new employee|Создать сотрудника|Қызметкер жасау
Link existing employee|Связать с сотрудником|Қызметкерге байланыстыру
Learning catalog|Каталог обучения|Оқу каталогы
Events and courses|Мероприятия и курсы|Іс-шаралар мен курстар
Review activities and see the strongest eligible candidates.|Просматривайте активности и подходящих кандидатов.|Іс-шаралар мен лайықты үміткерлерді қараңыз.
Search events…|Поиск мероприятий…|Іс-шараларды іздеу…
All types|Все типы|Барлық түрлер
Organization signals|Показатели организации|Ұйым көрсеткіштері
Development analytics|Аналитика развития|Даму талдауы
High-level workforce and learning catalog indicators.|Общие показатели сотрудников и каталога обучения.|Қызметкерлер мен оқу каталогының жалпы көрсеткіштері.
Employees by role|Сотрудники по ролям|Лауазымдар бойынша қызметкерлер
Current workforce distribution|Распределение сотрудников|Қызметкерлердің бөлінуі
Account|Аккаунт|Аккаунт
HR profile|Профиль HR|HR профилі
HR Dashboard|Кабинет HR|HR кабинеті
Your authenticated workspace identity.|Данные вашего аккаунта.|Аккаунтыңыздың деректері.
Employee opportunities|Возможности сотрудника|Қызметкер мүмкіндіктері
Recommended activities|Рекомендуемые активности|Ұсынылған іс-шаралар
Eligible activities ranked for the selected employee.|Подходящие активности для выбранного сотрудника.|Таңдалған қызметкерге лайықты іс-шаралар.
All formats|Все форматы|Барлық форматтар
Online|Онлайн|Онлайн
Offline|Очно|Офлайн
Self-paced|В своём темпе|Өз қарқынымен
Self Paced|В своём темпе|Өз қарқынымен
self_paced|В своём темпе|Өз қарқынымен
Capability map|Карта навыков|Дағдылар картасы
Employee skill gaps|Пробелы в навыках сотрудника|Қызметкердің дағды жетіспеушіліктері
Edit target|Изменить цель|Мақсатты өзгерту
Required for the target role.|Необходимо для целевой роли.|Мақсатты лауазымға қажет.
Additional readiness gaps.|Дополнительные навыки для развития.|Дамытуға арналған қосымша дағдылар.
Explainable guidance|Понятные рекомендации|Түсінікті ұсыныстар
Recommendation explanation|Объяснение рекомендации|Ұсыныстың түсіндірмесі
Inspect the evidence behind a recommendation.|Узнайте, почему выбрана рекомендация.|Ұсыныстың неге таңдалғанын біліңіз.
Deterministic mode|Режим на основе правил|Ережелерге негізделген режим
Grounded in employee data|На основе данных сотрудника|Қызметкер деректеріне негізделген
Ask about this employee’s path…|Спросите о карьерном пути сотрудника…|Қызметкердің мансап жолы туралы сұраңыз…
Update employee target|Обновить цель сотрудника|Қызметкердің мақсатын жаңарту
This immediately recalculates recommendations.|Рекомендации пересчитаются сразу.|Ұсыныстар бірден қайта есептеледі.
Event targeting|Подбор участников|Қатысушыларды іріктеу
Best candidates|Подходящие кандидаты|Лайықты үміткерлер
Eligible employees ranked by recommendation score.|Подходящие сотрудники по оценке соответствия.|Сәйкестік бағасы бойынша лайықты қызметкерлер.
Employee record|Карточка сотрудника|Қызметкер карточкасы
Create employee|Создать сотрудника|Қызметкер жасау
Missing optional details can remain empty.|Необязательные поля можно оставить пустыми.|Міндетті емес өрістерді бос қалдыруға болады.
Next ID generated if empty|Оставьте пустым для автоматического ID|Автоматты ID үшін бос қалдырыңыз
Job role|Должность|Лауазым
Junior|Начальный|Бастапқы
Middle|Средний|Орта
Senior|Старший|Аға
Lead|Ведущий|Жетекші
L&D Dashboard|Обзор обучения и развития|Оқыту және дамыту шолуы
Courses & Events|Курсы и мероприятия|Курстар мен іс-шаралар
Create event|Создать мероприятие|Іс-шара жасау
Create activity|Создать активность|Іс-шара жасау
Edit activity|Изменить активность|Іс-шараны өзгерту
Upcoming Sessions|Ближайшие занятия|Алдағы сабақтар
Course Analytics|Аналитика курсов|Курстар талдауы
L&D Profile|Профиль обучения и развития|Оқыту және дамыту профилі
Learning & Development|Обучение и развитие|Оқыту және дамыту
L&D Specialist|Специалист по обучению|Оқыту маманы
Catalog overview|Обзор каталога|Каталогқа шолу
Learning that closes real gaps|Обучение для нужных навыков|Қажетті дағдыларды дамытатын оқу
Manage development metadata while training continues in your existing tools.|Управляйте программами развития и ссылками на обучение.|Даму бағдарламалары мен оқу сілтемелерін басқарыңыз.
Next scheduled learning opportunities|Ближайшие запланированные занятия|Алдағы жоспарланған сабақтар
Catalog mix|Состав каталога|Каталог құрамы
Activities by delivery format|Активности по формату обучения|Оқу форматы бойынша іс-шаралар
Activity catalog|Каталог активностей|Іс-шаралар каталогы
Create and maintain career-development metadata and external learning links.|Создавайте активности и добавляйте ссылки на обучение.|Іс-шаралар жасап, оқу сілтемелерін қосыңыз.
Search title or skill…|Поиск по названию или навыку…|Атауы немесе дағдысы бойынша іздеу…
Type / Format|Тип / Формат|Түрі / Форматы
Target|Цель|Мақсат
Duration|Длительность|Ұзақтығы
Duration (hours)|Длительность (часы)|Ұзақтығы (сағат)
Sessions|Занятия|Сабақтар
Delivery calendar|Календарь занятий|Сабақтар күнтізбесі
Scheduled sessions across all externally delivered activities.|Расписание занятий по всем внешним программам.|Барлық сыртқы бағдарламалар бойынша сабақ кестесі.
Course performance|Результаты курсов|Курс нәтижелері
Participation and completion signals without exposing full employee profiles.|Участие и завершение обучения без доступа к полным профилям.|Толық профильдерді ашпай, қатысу мен аяқталу көрсеткіштері.
Your role is scoped to learning activity management.|Ваша роль — управление обучением.|Сіздің рөліңіз — оқуды басқару.
Activity metadata|Данные активности|Іс-шара деректері
Training remains external; Career Quest manages targeting and career impact.|Обучение проходит на внешних платформах; Career Quest управляет развитием.|Оқу сыртқы платформаларда өтеді; Career Quest дамуды басқарады.
Title|Название|Атауы
Description|Описание|Сипаттамасы
Type|Тип|Түрі
Format|Формат|Формат
Mandatory activity|Обязательная активность|Міндетті іс-шара
Learning link|Ссылка на обучение|Оқу сілтемесі
Skills developed|Развиваемые навыки|Дамытылатын дағдылар
Add skill|Добавить навык|Дағды қосу
Prerequisites|Предварительные требования|Алдын ала талаптар
Add prerequisite|Добавить требование|Талап қосу
One ISO date per line. Leave empty for self-paced activities.|Одна дата ГГГГ-ММ-ДД на строку. Для самостоятельного обучения оставьте пустым.|Әр жолға ЖЖЖЖ-АА-КК түріндегі бір күн. Өз қарқынымен оқу үшін бос қалдырыңыз.
Save activity|Сохранить активность|Іс-шараны сақтау
Course result|Результат курса|Курс нәтижесі
Assess participant|Оценить участника|Қатысушыны бағалау
Result|Результат|Нәтиже
Passed|Пройдено|Өткен
Failed|Не пройдено|Өтпеген
Score (optional)|Балл (необязательно)|Ұпай (міндетті емес)
Score|Балл|Ұпай
Feedback|Обратная связь|Кері байланыс
Save assessment|Сохранить оценку|Бағаны сақтау
Course|Курс|Курс
Workshop|Практикум|Практикалық сабақ
Mentoring|Менторство|Тәлімгерлік
Certification|Сертификация|Сертификаттау
Meetup|Встреча|Кездесу
Compliance|Обязательные нормы|Міндетті нормалар
Onboarding|Адаптация|Бейімдеу
Career readiness|Готовность к новой роли|Жаңа лауазымға дайындық
Choose a goal|Выберите цель|Мақсат таңдаңыз
Goal needed|Нужна цель|Мақсат қажет
Set my goal|Поставить цель|Мақсат қою
Set a goal|Поставьте цель|Мақсат қойыңыз
Set your career goal|Поставьте карьерную цель|Мансаптық мақсат қойыңыз
Choose a destination|Выберите направление|Бағытты таңдаңыз
Now|Сейчас|Қазір
Goal|Цель|Мақсат
Critical|Критический|Маңызды
critical blockers|критических препятствий|негізгі кедергі
Critical requirements met|Критические требования выполнены|Маңызды талаптар орындалды
No critical blockers|Критических препятствий нет|Негізгі кедергілер жоқ
No eligible quest right now|Пока нет подходящих заданий|Әзірге лайықты тапсырмалар жоқ
No recommendations|Нет рекомендаций|Ұсыныстар жоқ
Nothing in progress|Нет текущих активностей|Орындалып жатқан іс-шаралар жоқ
No courses found|Курсы не найдены|Курстар табылмады
Try updating your career goal or check back after new activities are published.|Измените карьерную цель или дождитесь новых активностей.|Мансаптық мақсатты өзгертіңіз немесе жаңа іс-шараларды күтіңіз.
You meet all critical requirements for this target.|Все критические требования для этой цели выполнены.|Осы мақсаттың барлық маңызды талаптары орындалды.
Choose a recommended activity when you are ready.|Выберите рекомендуемую активность, когда будете готовы.|Дайын болғанда ұсынылған іс-шараны таңдаңыз.
There are no eligible activities at the moment.|Сейчас нет подходящих активностей.|Қазір лайықты іс-шаралар жоқ.
Nothing is recorded in this section yet.|В этом разделе пока нет записей.|Бұл бөлімде әзірге жазбалар жоқ.
Try a different title, skill, role or grade.|Попробуйте другое название, навык, роль или уровень.|Басқа атау, дағды, лауазым немесе деңгейді қолданып көріңіз.
No critical blockers remain.|Критических препятствий больше нет.|Негізгі кедергілер қалған жоқ.
Growth requirements met|Требования для роста выполнены|Өсу талаптары орындалды
No other gaps remain.|Других пробелов нет.|Басқа жетіспеушіліктер жоқ.
Choose a career goal first|Сначала выберите карьерную цель|Алдымен мансаптық мақсатты таңдаңыз
Your visual path will appear here.|Здесь появится ваш карьерный путь.|Мансап жолыңыз осы жерде көрсетіледі.
Unlock readiness, focused gaps, and activities matched to where you want to go.|Узнайте готовность, недостающие навыки и подходящие активности.|Дайындығыңызды, жетіспейтін дағдыларды және лайықты іс-шараларды біліңіз.
Could not load your workspace.|Не удалось загрузить кабинет.|Кабинетті жүктеу мүмкін болмады.
Set a target to calculate readiness.|Задайте цель для расчёта готовности.|Дайындықты есептеу үшін мақсат қойыңыз.
No eligible candidates|Нет подходящих кандидатов|Лайықты үміткерлер жоқ
Loading eligible candidates…|Загрузка подходящих кандидатов…|Лайықты үміткерлер жүктелуде…
No employee currently meets the event filters and target gaps.|Сейчас нет сотрудников, подходящих под требования активности.|Қазір іс-шара талаптарына сай қызметкерлер жоқ.
Organization development access|Доступ к развитию сотрудников|Қызметкерлерді дамытуға қолжетімділік
Course and event management access|Доступ к управлению курсами|Курстарды басқаруға қолжетімділік
This role cannot open private employee profiles or the HR directory.|Эта роль не имеет доступа к личным профилям и каталогу HR.|Бұл рөлге жеке профильдер мен HR каталогы қолжетімсіз.
No activities in this status.|Нет активностей с таким статусом.|Бұл күйдегі іс-шаралар жоқ.
Levels to gain|Уровней до цели|Мақсатқа дейінгі деңгейлер
Recommended next quest|Следующее рекомендуемое задание|Келесі ұсынылған тапсырма
Product Discovery|Исследование продукта|Өнімді зерттеу
Roadmapping & Prioritization|Планирование и приоритизация|Жоспарлау және басымдықтарды анықтау
Product Analytics|Продуктовая аналитика|Өнім талдауы
UX Research|Исследование опыта пользователей|Пайдаланушы тәжірибесін зерттеу
Requirements Writing|Описание требований|Талаптарды жазу
Agile Practices|Гибкие методы работы|Икемді жұмыс әдістері
Talent Acquisition|Подбор персонала|Қызметкерлерді іріктеу
Employee Relations|Отношения с сотрудниками|Қызметкерлермен қарым-қатынас
Labor Law|Трудовое право|Еңбек құқығы
HR Analytics|HR-аналитика|HR талдауы
Learning Program Design|Разработка программ обучения|Оқу бағдарламаларын әзірлеу
Compensation & Benefits|Оплата труда и льготы|Еңбекақы және жеңілдіктер
Prospecting|Поиск клиентов|Клиенттерді іздеу
Negotiation|Переговоры|Келіссөздер
CRM Systems|CRM-системы|CRM жүйелері
Account Management|Работа с клиентами|Клиенттермен жұмыс
Product Knowledge|Знание продукта|Өнімді білу
Customer Service|Обслуживание клиентов|Клиенттерге қызмет көрсету
Technical Troubleshooting|Решение технических проблем|Техникалық мәселелерді шешу
BI Tools|Инструменты бизнес-аналитики|Бизнес талдау құралдары
Open training|Открыть обучение|Оқуды ашу
Why this quest?|Почему это задание?|Неліктен осы тапсырма?
Training link not added yet|Ссылка на обучение пока не добавлена|Оқу сілтемесі әлі қосылмаған
View learning path|Посмотреть путь обучения|Оқу жолын көру
View full learning path|Полный путь обучения|Толық оқу жолы
Your suggested learning order|Рекомендуемый порядок обучения|Ұсынылған оқу реті
Start with your learning path|Начните с пути обучения|Оқу жолынан бастаңыз
The first step builds prerequisites for later activities.|Первый шаг готовит к следующим активностям.|Бірінші қадам келесі іс-шараларға дайындайды.
You are here|Вы здесь|Сіз осындасыз
Your goal|Ваша цель|Сіздің мақсатыңыз
Current role target|Требования текущей роли|Ағымдағы лауазым талаптары
Your current context|Ваши текущие данные|Сіздің ағымдағы деректеріңіз
Private to your employee account.|Доступно только в вашем кабинете.|Тек сіздің кабинетіңізде қолжетімді.
Readiness|Готовность|Дайындық
Top quest|Следующее задание|Келесі тапсырма
None available|Нет доступных|Қолжетімді емес
Work format|Формат работы|Жұмыс форматы
Last skill review|Последняя оценка навыков|Дағдыларды соңғы бағалау
Not set|Не задано|Белгіленбеген
Career goal updated|Карьерная цель обновлена|Мансаптық мақсат жаңартылды
Session expired|Сессия завершена|Сессия аяқталды
Checking your career data|Проверяем ваши карьерные данные|Мансаптық деректеріңіз тексерілуде
Participants|Участники|Қатысушылар
Activity records|Записи активности|Іс-шара жазбалары
Average completion|Среднее завершение|Орташа аяқталу
Completions|Завершения|Аяқтаулар
Status distribution|Распределение статусов|Күйлердің бөлінуі
Not assessed|Не оценено|Бағаланбаған
Assess|Оценить|Бағалау
No enrolled participants.|Записанных участников нет.|Тіркелген қатысушылар жоқ.
Gain|Прирост|Өсім
Max level|Максимальный уровень|Ең жоғары деңгей
Minimum level|Минимальный уровень|Ең төменгі деңгей
Catalog activities|Активности в каталоге|Каталогтағы іс-шаралар
Optional development|Добровольное развитие|Ерікті даму
External links configured|Добавленные ссылки|Қосылған сілтемелер
Voluntary activities|Добровольные активности|Ерікті іс-шаралар
Career goals set|Карьерные цели заданы|Мансаптық мақсаттар қойылған
No sessions|Нет занятий|Сабақтар жоқ
Add dates to scheduled activities.|Добавьте даты занятий.|Сабақ күндерін қосыңыз.
Activity updated|Активность обновлена|Іс-шара жаңартылды
Activity created|Активность создана|Іс-шара жасалды
Employee created|Сотрудник создан|Қызметкер жасалды
Registration rejected|Заявка отклонена|Өтінім қабылданбады
Assessment saved and skills awarded|Оценка сохранена, навыки обновлены|Баға сақталды, дағдылар жаңартылды
Assessment saved; no skills awarded|Оценка сохранена без начисления навыков|Баға сақталды, дағдылар қосылмады
Backend Engineer|Бэкенд-разработчик|Бэкенд әзірлеуші
Frontend Engineer|Фронтенд-разработчик|Фронтенд әзірлеуші
Data Analyst|Аналитик данных|Деректер талдаушысы
QA Engineer|Инженер по тестированию|Тестілеу инженері
Product Manager|Менеджер продукта|Өнім менеджері
HR Business Partner|HR бизнес-партнёр|HR бизнес-серіктес
Sales Manager|Менеджер по продажам|Сату менеджері
Customer Support Specialist|Специалист поддержки|Қолдау маманы
API Design|Проектирование API|API жобалау
System Design|Проектирование систем|Жүйелерді жобалау
Cloud Platforms|Облачные платформы|Бұлттық платформалар
Containers & Orchestration|Контейнеры и оркестрация|Контейнерлер және оркестрация
Application Security|Безопасность приложений|Қолданбалар қауіпсіздігі
Observability|Наблюдаемость систем|Жүйелерді бақылау
Web Performance|Производительность веба|Веб өнімділігі
Web Accessibility|Веб-доступность|Веб қолжетімділігі
Test Design|Проектирование тестов|Тесттерді жобалау
Test Automation|Автоматизация тестирования|Тестілеуді автоматтандыру
API Testing|Тестирование API|API тестілеу
Load Testing|Нагрузочное тестирование|Жүктемені тестілеу
Statistics|Статистика|Статистика
Data Visualization|Визуализация данных|Деректерді визуализациялау
Data Modeling|Моделирование данных|Деректерді модельдеу
Machine Learning Fundamentals|Основы машинного обучения|Машиналық оқыту негіздері
Project Management|Управление проектами|Жобаларды басқару
Communication|Коммуникация|Қарым-қатынас
Public Speaking|Публичные выступления|Көпшілік алдында сөйлеу
Written Communication|Письменная коммуникация|Жазбаша қарым-қатынас
Stakeholder Management|Работа с заинтересованными сторонами|Мүдделі тараптармен жұмыс
Leadership|Лидерство|Көшбасшылық
Conflict Resolution|Разрешение конфликтов|Қақтығыстарды шешу
Teamwork|Командная работа|Топтық жұмыс
Emotional Intelligence|Эмоциональный интеллект|Эмоциялық интеллект
Problem Solving|Решение проблем|Мәселелерді шешу
Critical Thinking|Критическое мышление|Сыни ойлау
Time Management|Управление временем|Уақытты басқару
Adaptability|Адаптивность|Бейімделгіштік
Welcome back, {0}|С возвращением, {0}|Қайта қош келдіңіз, {0}
Step {0}|Шаг {0}|{0}-қадам
Start here|Начните здесь|Осы жерден бастаңыз
After earlier steps|После предыдущих шагов|Алдыңғы қадамдардан кейін
{0} study hours|{0} часов обучения|{0} сағат оқу
Projected readiness: {0}|Прогноз готовности: {0}|Болжамды дайындық: {0}
Projected readiness after step: {0}|Готовность после шага: {0}|Қадамнан кейінгі дайындық: {0}
{0} skill gaps projected to remain|Останется пробелов в навыках: {0}|Қалатын дағды жетіспеушіліктері: {0}
Level {0}|Уровень {0}|Деңгей {0}
Target {0}|Цель {0}|Мақсат {0}
{0} match|Соответствие: {0}|Сәйкестік: {0}
{0} gaps in total|Всего пробелов: {0}|Барлық жетіспеушілік: {0}
{0} steps toward {1}|{0} шага к цели {1}|{1} мақсатына {0} қадам
0 · No knowledge|0 · Нет знаний|0 · Білімі жоқ
5 · Expert|5 · Эксперт|5 · Сарапшы
`;
  const dictionary = new Map();
  const patterns = [];
  for (const line of rows.trim().split('\n')) {
    const [en, ru, kk] = line.split('|');
    if (!en || !ru || !kk) continue;
    dictionary.set(en.toLowerCase(), {ru, kk});
    if (en.includes('{')) {
      const regex = en.split(/(\{\d+\})/).map(part => /^\{\d+\}$/.test(part) ? '(.+?)' : part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('');
      patterns.push({regex: new RegExp('^' + regex + '$', 'i'), ru, kk});
    }
  }
  function translate(source, language) {
    if (language === 'en' || !['ru', 'kk'].includes(language)) return source;
    const text = source.trim();
    const entry = dictionary.get(text.toLowerCase());
    let translated;
    if (entry) {
      translated = entry[language];
      if (/[A-Z]/.test(text) && text === text.toUpperCase()) translated = translated.toLocaleUpperCase();
    } else if (text.includes(' · ')) {
      translated = text.split(' · ').map(part => translate(part, language)).join(' · ');
    } else {
      for (const pattern of patterns) {
        const match = text.match(pattern.regex);
        if (match) {
          translated = pattern[language].replace(/\{(\d+)\}/g, (_, i) => translate(match[Number(i) + 1], language));
          break;
        }
      }
      if (!translated && text.includes(' · ')) translated = text.split(' · ').map(part => translate(part, language)).join(' · ');
      if (!translated) {
        const decorated = text.match(/^([＋+↪\s]*)(.*?)([→↗\s]*)$/);
        if (decorated && decorated[2] !== text && dictionary.has(decorated[2].toLowerCase())) translated = decorated[1] + translate(decorated[2], language) + decorated[3];
      }
    }
    return translated ? source.replace(text, translated) : source;
  }
  function init() {
    let language = 'en';
    try { language = localStorage.getItem('careerquest.language') || 'en'; } catch (_) {}
    if (!['en','ru','kk'].includes(language)) language = 'en';
    const records = new WeakMap();
    const excluded = 'script,style,textarea,code,pre,.brand,[data-no-i18n]';
    const switcher = document.createElement('label');
    switcher.className = 'language-switcher';
    switcher.setAttribute('data-no-i18n', '');
    switcher.innerHTML = '<span aria-hidden="true">◎</span><select aria-label="Language / Язык / Тіл"><option value="en">English</option><option value="ru">Русский</option><option value="kk">Қазақша</option></select>';
    const host = document.querySelector('.topbar') || document.querySelector('.login-box') || document.body;
    if (host.matches('.login-box')) host.prepend(switcher); else host.append(switcher);
    const select = switcher.querySelector('select');
    select.value = language;
    function replace(target, key, read, write) {
      const current = read();
      let record = records.get(target);
      if (!record) { record = {}; records.set(target, record); }
      if (!record[key] || record[key].rendered !== current) record[key] = {source: current, rendered: current};
      const value = translate(record[key].source, language);
      if (current !== value) write(value);
      record[key].rendered = value;
    }
    const observer = new MutationObserver(() => render());
    function render() {
      observer.disconnect();
      // Options without explicit values otherwise change their submitted value when translated.
      document.querySelectorAll('option:not([value])').forEach(option => option.setAttribute('value', option.textContent));
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      while (walker.nextNode()) {
        const node = walker.currentNode;
        if (!node.parentElement || node.parentElement.closest(excluded) || !node.nodeValue.trim()) continue;
        replace(node, 'text', () => node.nodeValue, value => { node.nodeValue = value; });
      }
      document.querySelectorAll('[placeholder],[title],[aria-label]').forEach(element => {
        if (element.closest(excluded)) return;
        for (const attr of ['placeholder','title','aria-label']) if (element.hasAttribute(attr)) replace(element, attr, () => element.getAttribute(attr), value => element.setAttribute(attr, value));
      });
      replace(document, 'title', () => document.title, value => { document.title = value; });
      document.documentElement.lang = language;
      observer.observe(document.body, {subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ['placeholder','title','aria-label']});
    }
    select.addEventListener('change', () => {
      language = select.value;
      try { localStorage.setItem('careerquest.language', language); } catch (_) {}
      render();
    });
    render();
  }
  return {translate, init};
})();
if (typeof module !== 'undefined' && module.exports) module.exports = CareerQuestI18n;
if (typeof document !== 'undefined') document.addEventListener('DOMContentLoaded', CareerQuestI18n.init);
