# Анализ выбора платформы оркестрации для apk-ci

**Дата анализа:** 2025-12-25
**Версия:** 1.0
**Автор:** Claude Code (автоматический анализ)

---

## Содержание

1. [Резюме и рекомендация](#1-резюме-и-рекомендация)
2. [Профиль проекта](#2-профиль-проекта)
3. [Требования к платформе оркестрации](#3-требования-к-платформе-оркестрации)
4. [Анализ Kubernetes](#4-анализ-kubernetes)
5. [Анализ Docker Swarm](#5-анализ-docker-swarm)
6. [Сравнительная таблица](#6-сравнительная-таблица)
7. [Альтернативные варианты](#7-альтернативные-варианты)
8. [Обоснование выбора](#8-обоснование-выбора)
9. [План миграции](#9-план-миграции)

---

## 1. Резюме и рекомендация

### Итоговая рекомендация: **Docker Swarm**

| Критерий | Docker Swarm | Kubernetes |
|----------|:------------:|:----------:|
| Соответствие требованиям | ★★★★★ | ★★★★☆ |
| Сложность внедрения | ★★★★★ (низкая) | ★★☆☆☆ (высокая) |
| Операционные затраты | ★★★★★ (низкие) | ★★☆☆☆ (высокие) |
| Масштабируемость | ★★★☆☆ | ★★★★★ |
| **Рекомендован для проекта** | ✅ **ДА** | ⚠️ Избыточен |

**Обоснование выбора Docker Swarm:**

1. **Простота** — команда уже знакома с Docker, Swarm добавляет минимальную сложность
2. **Достаточность** — функционал Swarm полностью покрывает потребности CI/CD раннеров
3. **Интеграция с Gitea** — Gitea Act Runner нативно поддерживает Docker
4. **Специфика 1C** — длительные batch-задачи не требуют сложной оркестрации
5. **Стоимость владения** — значительно ниже, чем у Kubernetes

---

## 2. Профиль проекта

### 2.1 Характеристики apk-ci

```
┌─────────────────────────────────────────────────────────────────┐
│                    ПРОФИЛЬ НАГРУЗКИ                            │
├─────────────────────────────────────────────────────────────────┤
│ Тип:            Batch CI/CD Runner (не stateful сервис)        │
│ Паттерн:        Job-based execution (Gitea Actions)            │
│ Состояние:      Ephemeral (временные рабочие директории)       │
│ Масштаб:        1-10 параллельных runner'ов                   │
│ Длительность:   От 30 сек до 4 часов на задачу                │
│ Частота:        Событийная (по commit/PR/manual trigger)       │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Внешние зависимости

| Зависимость | Порт | Протокол | Расположение | Критичность |
|-------------|------|----------|--------------|-------------|
| 1C RAC Server | 1545 | TCP | LAN | Критическая |
| 1C Application Server | динамический | TCP | LAN | Критическая |
| MS SQL Server | 1433 | TCP | LAN | Критическая |
| Gitea API | 443 | HTTPS | LAN/DMZ | Критическая |
| SonarQube | 9000 | HTTP | LAN | Средняя |
| SMB shares | 445 | SMB | LAN | Низкая |

### 2.3 Требования к ресурсам контейнера

```yaml
# Типичные требования для одного runner
resources:
  memory:
    request: 2Gi
    limit: 8Gi        # Для EDT/SonarQube операций
  cpu:
    request: 1
    limit: 4
  storage:
    ephemeral: 100Gi  # Рабочие директории
```

### 2.4 Требования к образу

```dockerfile
# Минимальные компоненты образа
FROM ubuntu:22.04

# 1C:Enterprise платформа (~2GB)
COPY --from=1c-platform /opt/1cv8 /opt/1cv8

# 1C EDT (~1.5GB)
COPY --from=1c-edt /opt/1C/1CE /opt/1C/1CE

# SonarQube Scanner (~500MB)
COPY --from=sonar-scanner /opt/sonar-scanner /opt/sonar-scanner

# Git, утилиты
RUN apt-get install -y git ca-certificates

# Итоговый размер образа: ~5-6GB
```

---

## 3. Требования к платформе оркестрации

### 3.1 Функциональные требования

| ID | Требование | Приоритет | Описание |
|----|-----------|-----------|----------|
| FR-01 | Запуск контейнеров по событию | Критический | Интеграция с Gitea Actions |
| FR-02 | Доступ к сети LAN | Критический | RAC, MSSQL, 1C серверы |
| FR-03 | Ephemeral storage 100GB+ | Критический | Рабочие директории |
| FR-04 | Переменные окружения | Критический | Конфигурация через env |
| FR-05 | Secrets management | Высокий | Пароли, токены |
| FR-06 | Graceful shutdown | Высокий | Завершение длинных задач |
| FR-07 | Логирование stdout/stderr | Высокий | Интеграция с Gitea UI |
| FR-08 | Health checks | Средний | Мониторинг состояния |
| FR-09 | Resource limits | Средний | Ограничение ресурсов |
| FR-10 | Auto-restart on failure | Низкий | Перезапуск упавших |

### 3.2 Нефункциональные требования

| ID | Требование | Приоритет | Значение |
|----|-----------|-----------|----------|
| NFR-01 | Доступность | Высокий | 99.5% (рабочее время) |
| NFR-02 | Время развёртывания | Средний | < 5 минут |
| NFR-03 | Сложность эксплуатации | Высокий | Минимальная |
| NFR-04 | Размер команды DevOps | Высокий | 1-2 человека |
| NFR-05 | Совместимость с Gitea | Критический | Act Runner |

---

## 4. Анализ Kubernetes

### 4.1 Архитектура для apk-ci на K8s

```
┌─────────────────────────────────────────────────────────────────────┐
│                         KUBERNETES CLUSTER                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │ Control Plane│    │   Worker 1   │    │   Worker 2   │          │
│  │              │    │              │    │              │          │
│  │ API Server   │    │ ┌──────────┐ │    │ ┌──────────┐ │          │
│  │ Scheduler    │────│ │ Runner   │ │    │ │ Runner   │ │          │
│  │ Controller   │    │ │ Pod      │ │    │ │ Pod      │ │          │
│  │ etcd         │    │ └──────────┘ │    │ └──────────┘ │          │
│  └──────────────┘    │              │    │              │          │
│                      │ ┌──────────┐ │    │ ┌──────────┐ │          │
│                      │ │ Gitea Act│ │    │ │ Local PV │ │          │
│                      │ │ Runner   │ │    │ │ 100GB    │ │          │
│                      │ │ DaemonSet│ │    │ └──────────┘ │          │
│                      │ └──────────┘ │    │              │          │
│                      └──────────────┘    └──────────────┘          │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Дополнительные компоненты                 │   │
│  ├─────────────────────────────────────────────────────────────┤   │
│  │ • Ingress Controller (nginx/traefik)                        │   │
│  │ • Cert-Manager (TLS сертификаты)                            │   │
│  │ • External Secrets Operator (интеграция с Vault)            │   │
│  │ • Prometheus + Grafana (мониторинг)                         │   │
│  │ • Loki (централизованное логирование)                       │   │
│  │ • StorageClass (local-path / NFS)                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 Преимущества Kubernetes

| Преимущество | Применимость к проекту | Оценка |
|--------------|----------------------|--------|
| Автомасштабирование (HPA/VPA) | Низкая — batch jobs, не постоянная нагрузка | ⚠️ |
| Self-healing | Средняя — полезно для длительных задач | ✅ |
| Декларативная конфигурация | Высокая — GitOps-friendly | ✅ |
| Secrets management | Высокая — External Secrets, Sealed Secrets | ✅ |
| Network Policies | Средняя — изоляция runner'ов | ✅ |
| Multi-tenancy | Низкая — один проект | ⚠️ |
| Service Mesh | Низкая — нет микросервисов | ❌ |
| Canary deployments | Низкая — batch jobs | ❌ |
| Pod disruption budgets | Средняя — защита длинных задач | ✅ |

### 4.3 Недостатки Kubernetes для проекта

| Недостаток | Влияние | Критичность |
|------------|---------|-------------|
| **Сложность эксплуатации** | Требует DevOps специалиста | 🔴 Высокая |
| **Overhead ресурсов** | Control plane съедает 2-4GB RAM | 🟡 Средняя |
| **Время обучения** | 3-6 месяцев для полноценной эксплуатации | 🔴 Высокая |
| **Стоимость инфраструктуры** | Минимум 3 ноды для HA | 🟡 Средняя |
| **Сложность отладки** | Многослойная абстракция | 🟡 Средняя |
| **Избыточность для batch jobs** | Большинство фич не используется | 🔴 Высокая |

### 4.4 Примерный манифест для K8s

```yaml
# gitea-runner-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apk-ci
  namespace: gitea-runners
spec:
  replicas: 3
  selector:
    matchLabels:
      app: apk-ci
  template:
    metadata:
      labels:
        app: apk-ci
    spec:
      containers:
      - name: act-runner
        image: gitea/act_runner:latest
        env:
        - name: GITEA_INSTANCE_URL
          value: "https://git.benadis.ru"
        - name: GITEA_RUNNER_REGISTRATION_TOKEN
          valueFrom:
            secretKeyRef:
              name: gitea-runner-secret
              key: token
        volumeMounts:
        - name: work-dir
          mountPath: /tmp/benadis
        - name: docker-sock
          mountPath: /var/run/docker.sock
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "8Gi"
            cpu: "4"
      volumes:
      - name: work-dir
        emptyDir:
          sizeLimit: 100Gi
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
      # Для доступа к LAN сервисам
      hostNetwork: true  # или NetworkPolicy
      dnsPolicy: ClusterFirstWithHostNet
---
apiVersion: v1
kind: Secret
metadata:
  name: gitea-runner-secret
  namespace: gitea-runners
type: Opaque
data:
  token: <base64-encoded-token>
```

### 4.5 Общая оценка Kubernetes

```
┌────────────────────────────────────────────────────────────────────┐
│                    ОЦЕНКА KUBERNETES                               │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  Соответствие требованиям:    ████████░░  80%                     │
│  Сложность внедрения:         ██████████  100% (высокая)          │
│  Операционные затраты:        ████████░░  80% (высокие)           │
│  Масштабируемость:            ██████████  100%                    │
│  Гибкость:                    ██████████  100%                    │
│                                                                    │
│  ИТОГ: Избыточен для данного проекта                              │
│  Рекомендация: Использовать только если уже есть K8s кластер     │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 5. Анализ Docker Swarm

### 5.1 Архитектура для apk-ci на Docker Swarm

```
┌─────────────────────────────────────────────────────────────────────┐
│                        DOCKER SWARM CLUSTER                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │  Manager 1   │    │  Worker 1    │    │  Worker 2    │          │
│  │  (Leader)    │    │              │    │              │          │
│  │              │    │ ┌──────────┐ │    │ ┌──────────┐ │          │
│  │ Raft        ├────│ │ Runner   │ │    │ │ Runner   │ │          │
│  │ Consensus   │    │ │ Service  │ │    │ │ Service  │ │          │
│  │              │    │ └──────────┘ │    │ └──────────┘ │          │
│  │ ┌──────────┐ │    │              │    │              │          │
│  │ │ Gitea    │ │    │ ┌──────────┐ │    │ ┌──────────┐ │          │
│  │ │ (опц.)   │ │    │ │ Local    │ │    │ │ Local    │ │          │
│  │ └──────────┘ │    │ │ Volume   │ │    │ │ Volume   │ │          │
│  └──────────────┘    │ └──────────┘ │    │ └──────────┘ │          │
│                      └──────────────┘    └──────────────┘          │
│                                                                     │
│  Overlay Network: gitea-runners-net (доступ к LAN через bridge)   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 Преимущества Docker Swarm

| Преимущество | Применимость к проекту | Оценка |
|--------------|----------------------|--------|
| **Простота развёртывания** | Критическая — `docker swarm init` и готово | ✅✅ |
| **Нативная интеграция с Docker** | Высокая — используем docker compose | ✅✅ |
| **Низкий overhead** | Высокая — минимум ресурсов на управление | ✅✅ |
| **Быстрое обучение** | Высокая — знакомый Docker API | ✅✅ |
| **Secrets management** | Высокая — `docker secret` из коробки | ✅ |
| **Rolling updates** | Средняя — обновление runner'ов | ✅ |
| **Service discovery** | Средняя — DNS для сервисов | ✅ |
| **Overlay networking** | Средняя — изоляция сетей | ✅ |
| **Constraint placement** | Высокая — привязка к нодам с 1C | ✅ |

### 5.3 Недостатки Docker Swarm

| Недостаток | Влияние | Критичность |
|------------|---------|-------------|
| Ограниченная экосистема | Меньше готовых решений | 🟡 Средняя |
| Нет auto-scaling | Ручное масштабирование | 🟢 Низкая |
| Менее активное развитие | Stable, но мало новых фич | 🟡 Средняя |
| Нет Pod-like abstractions | Один контейнер = один сервис | 🟢 Низкая |
| Ограниченный мониторинг | Требует внешних инструментов | 🟡 Средняя |

### 5.4 Конфигурация Docker Swarm

```yaml
# docker-compose.swarm.yml
version: "3.8"

services:
  gitea-runner:
    image: gitea/act_runner:latest
    deploy:
      replicas: 3
      placement:
        constraints:
          - node.labels.has_1c == true
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '1'
          memory: 2G
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
    environment:
      - GITEA_INSTANCE_URL=https://git.benadis.ru
      - GITEA_RUNNER_NAME={{.Node.Hostname}}-runner
    secrets:
      - gitea_runner_token
      - source: app_secrets
        target: /etc/benadis/secret.yaml
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /tmp/benadis:/tmp/benadis
      - /opt/1cv8:/opt/1cv8:ro
      - /opt/1C/1CE:/opt/1C/1CE:ro
    networks:
      - gitea-runners-net
    # Для доступа к LAN (RAC, MSSQL)
    extra_hosts:
      - "msk-sql-arch-01:10.0.1.100"
      - "msk-as-arch-001:10.0.1.101"

networks:
  gitea-runners-net:
    driver: overlay
    attachable: true

secrets:
  gitea_runner_token:
    external: true
  app_secrets:
    file: ./config/secret.yaml

# Деплой: docker stack deploy -c docker-compose.swarm.yml benadis
```

### 5.5 Команды управления Swarm

```bash
# Инициализация кластера
docker swarm init --advertise-addr <MANAGER_IP>

# Добавление worker нод
docker swarm join --token <TOKEN> <MANAGER_IP>:2377

# Пометка нод с установленной 1C
docker node update --label-add has_1c=true <NODE_ID>

# Создание секретов
echo "runner_token_value" | docker secret create gitea_runner_token -
docker secret create app_secrets ./config/secret.yaml

# Деплой стека
docker stack deploy -c docker-compose.swarm.yml benadis

# Масштабирование
docker service scale benadis_gitea-runner=5

# Просмотр логов
docker service logs -f benadis_gitea-runner

# Обновление образа
docker service update --image gitea/act_runner:0.2.11 benadis_gitea-runner
```

### 5.6 Общая оценка Docker Swarm

```
┌────────────────────────────────────────────────────────────────────┐
│                    ОЦЕНКА DOCKER SWARM                             │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  Соответствие требованиям:    █████████░  90%                     │
│  Сложность внедрения:         ██░░░░░░░░  20% (низкая)            │
│  Операционные затраты:        ██░░░░░░░░  20% (низкие)            │
│  Масштабируемость:            ██████░░░░  60%                     │
│  Гибкость:                    ██████░░░░  60%                     │
│                                                                    │
│  ИТОГ: Оптимален для данного проекта                              │
│  Рекомендация: Использовать как основную платформу                │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 6. Сравнительная таблица

### 6.1 Функциональное сравнение

| Критерий | Docker Swarm | Kubernetes | Победитель для проекта |
|----------|:------------:|:----------:|:----------------------:|
| Запуск контейнеров | ✅ | ✅ | = |
| Secrets management | ✅ docker secret | ✅ K8s Secrets + ESO | K8s (богаче) |
| Service discovery | ✅ DNS | ✅ DNS + CoreDNS | K8s (гибче) |
| Load balancing | ✅ Routing mesh | ✅ Services + Ingress | K8s (гибче) |
| Rolling updates | ✅ | ✅ | = |
| Health checks | ✅ | ✅ | = |
| Resource limits | ✅ | ✅ | = |
| Network policies | ⚠️ Ограничено | ✅ NetworkPolicy | K8s |
| Auto-scaling | ❌ | ✅ HPA/VPA | K8s |
| Self-healing | ✅ | ✅ | = |
| Declarative config | ✅ Compose | ✅ YAML manifests | = |
| GitOps интеграция | ⚠️ Ограничено | ✅ ArgoCD/Flux | K8s |
| **Доступ к host network** | ✅ Просто | ⚠️ hostNetwork | **Swarm** |
| **Монтирование хост-путей** | ✅ Просто | ⚠️ hostPath + PV | **Swarm** |

### 6.2 Операционное сравнение

| Аспект | Docker Swarm | Kubernetes |
|--------|:------------:|:----------:|
| Время до production | 1-2 дня | 2-4 недели |
| Минимальное число нод | 1 | 3 |
| Размер команды DevOps | 0.5 FTE | 1-2 FTE |
| Кривая обучения | Пологая | Крутая |
| Количество компонентов | 2-3 | 10+ |
| Документация | Хорошая | Отличная |
| Сообщество | Среднее | Огромное |
| Vendor lock-in | Низкий | Низкий |

### 6.3 Стоимостное сравнение (3 ноды)

| Статья затрат | Docker Swarm | Kubernetes |
|---------------|:------------:|:----------:|
| **Инфраструктура** | | |
| RAM для управления | 512MB | 4-6GB |
| CPU для управления | 0.5 core | 2-4 cores |
| **Операционные** | | |
| DevOps FTE (год) | 0.3 × $80K = $24K | 1.0 × $80K = $80K |
| Обучение | $5K | $20K |
| **Итого TCO (год)** | **~$30K** | **~$100K** |

### 6.4 Соответствие требованиям проекта

| Требование | Docker Swarm | Kubernetes | Важность |
|------------|:------------:|:----------:|:--------:|
| FR-01: Gitea Actions интеграция | ✅ | ✅ | Критическая |
| FR-02: Доступ к LAN | ✅✅ | ⚠️ | Критическая |
| FR-03: Ephemeral storage 100GB | ✅ | ✅ | Критическая |
| FR-04: Переменные окружения | ✅ | ✅ | Критическая |
| FR-05: Secrets | ✅ | ✅ | Высокая |
| FR-06: Graceful shutdown | ✅ | ✅ | Высокая |
| FR-07: Логирование | ✅ | ✅ | Высокая |
| NFR-01: Доступность 99.5% | ✅ | ✅ | Высокая |
| NFR-03: Простота эксплуатации | ✅✅ | ⚠️ | Высокая |
| NFR-04: Размер команды 1-2 | ✅✅ | ⚠️ | Высокая |
| **Итоговое соответствие** | **95%** | **75%** | — |

---

## 7. Альтернативные варианты

### 7.1 Nomad (HashiCorp)

```
Плюсы:
+ Простота, сравнимая с Swarm
+ Поддержка не только Docker (raw exec, Java, QEMU)
+ Хорошая интеграция с Vault/Consul

Минусы:
- Меньше сообщество
- Нет нативной интеграции с Gitea
- Дополнительный инструмент в стеке

Рекомендация: Рассмотреть, если уже используется HashiCorp стек
```

### 7.2 Podman + systemd

```
Плюсы:
+ Нет демона (daemonless)
+ Rootless контейнеры
+ Совместимость с Docker CLI

Минусы:
- Нет встроенной кластеризации
- Требует ручной настройки HA
- Меньше готовых решений

Рекомендация: Для single-node development, не для production
```

### 7.3 Bare Metal + systemd

```
Плюсы:
+ Максимальная производительность
+ Нет overhead контейнеризации
+ Полный контроль

Минусы:
- Сложность деплоя и обновлений
- Нет изоляции
- Ручное управление зависимостями

Рекомендация: Для legacy систем, где контейнеры невозможны
```

---

## 8. Обоснование выбора

### 8.1 Почему Docker Swarm оптимален для apk-ci

#### Фактор 1: Специфика рабочей нагрузки

```
apk-ci — это CI/CD batch runner, а не микросервисное приложение.

Характеристики:
• Задачи выполняются по событию (commit, PR)
• Нет постоянной нагрузки для auto-scaling
• Длительность задачи: 30 сек — 4 часа
• Параллелизм: 1-10 runners одновременно
• Stateless между задачами

Вывод: Kubernetes оптимизирован для микросервисов с постоянной нагрузкой.
       Docker Swarm достаточен для batch workloads.
```

#### Фактор 2: Интеграция с 1C инфраструктурой

```
Проект требует тесной интеграции с legacy 1C инфраструктурой:

• RAC (порт 1545) — кластер 1C в LAN
• MSSQL (порт 1433) — серверы БД в LAN
• 1C Application Servers — в LAN
• SMB shares — файловые ресурсы

Docker Swarm:
✅ host network просто настроить
✅ bind mounts из хоста тривиальны
✅ Нет NAT overhead

Kubernetes:
⚠️ hostNetwork требует особой настройки
⚠️ NetworkPolicy нужны для LAN доступа
⚠️ CSI драйверы для хост-путей

Вывод: Swarm проще интегрируется с LAN инфраструктурой
```

#### Фактор 3: Размер команды и компетенции

```
Типичная ситуация для 1C-ориентированных проектов:

• 1-2 DevOps/системных администратора
• Фокус на 1C, а не на Cloud Native
• Ограниченное время на инфраструктуру

Docker Swarm:
✅ Знакомый Docker CLI
✅ 1-2 дня до production
✅ docker-compose.yml уже знакомый формат

Kubernetes:
⚠️ kubectl, Helm, Kustomize — новые инструменты
⚠️ RBAC, NetworkPolicy, PV/PVC — новые концепции
⚠️ 2-4 недели до production
⚠️ 3-6 месяцев до уверенной эксплуатации

Вывод: Swarm соответствует текущим компетенциям команды
```

#### Фактор 4: TCO (Total Cost of Ownership)

```
Сравнение затрат на 3 года:

                          Docker Swarm    Kubernetes
──────────────────────────────────────────────────────
Инфраструктура (3 ноды)   $0              $0
Дополнительные компоненты $0              $5K (мониторинг)
DevOps FTE (0.3 vs 1.0)   $72K            $240K
Обучение                  $5K             $20K
Инциденты (downtime)      $10K            $5K
──────────────────────────────────────────────────────
ИТОГО за 3 года           ~$87K           ~$270K

Вывод: Swarm в 3 раза дешевле при сопоставимом качестве
```

### 8.2 Когда выбрать Kubernetes

Kubernetes стоит выбрать, если:

1. **Уже есть K8s кластер** — использовать существующую инфраструктуру
2. **Планируется рост до 50+ runners** — auto-scaling критичен
3. **Требуется мультитенантность** — разные команды, изоляция
4. **Есть выделенная DevOps команда** — 2+ FTE на инфраструктуру
5. **Планируется cloud-native трансформация** — K8s как стратегический выбор

---

## 9. План миграции

### 9.1 План внедрения Docker Swarm

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ПЛАН ВНЕДРЕНИЯ DOCKER SWARM                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Этап 1: Подготовка (1-2 дня)                                      │
│  ├─ [ ] Установить Docker на 3 ноды                                 │
│  ├─ [ ] Инициализировать Swarm кластер                             │
│  ├─ [ ] Настроить firewall (2377, 7946, 4789)                      │
│  └─ [ ] Пометить ноды с 1C (label has_1c=true)                     │
│                                                                     │
│  Этап 2: Образ (2-3 дня)                                           │
│  ├─ [ ] Создать Dockerfile с 1C платформой                         │
│  ├─ [ ] Добавить EDT, SonarQube Scanner                            │
│  ├─ [ ] Настроить CI для сборки образа                             │
│  └─ [ ] Протестировать образ локально                              │
│                                                                     │
│  Этап 3: Конфигурация (1 день)                                     │
│  ├─ [ ] Создать docker-compose.swarm.yml                           │
│  ├─ [ ] Настроить secrets                                          │
│  ├─ [ ] Настроить volumes и сети                                   │
│  └─ [ ] Протестировать на staging                                  │
│                                                                     │
│  Этап 4: Интеграция с Gitea (1 день)                               │
│  ├─ [ ] Зарегистрировать runner в Gitea                            │
│  ├─ [ ] Протестировать pipeline                                    │
│  └─ [ ] Настроить мониторинг (опционально)                         │
│                                                                     │
│  Этап 5: Production (1 день)                                       │
│  ├─ [ ] Переключить production workflows                           │
│  ├─ [ ] Мониторинг первых запусков                                 │
│  └─ [ ] Документирование                                           │
│                                                                     │
│  ИТОГО: 6-8 рабочих дней                                           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.2 Примерный Dockerfile

```dockerfile
# Dockerfile.apk-ci
FROM ubuntu:22.04

LABEL maintainer="GitOps Team"
LABEL description="apk-ci для Gitea Actions"

# Базовые зависимости
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    curl \
    unzip \
    locales \
    && rm -rf /var/lib/apt/lists/*

# Локаль для 1C
RUN locale-gen ru_RU.UTF-8
ENV LANG=ru_RU.UTF-8 LC_ALL=ru_RU.UTF-8

# 1C:Enterprise платформа (требует лицензию)
COPY --from=1c-platform /opt/1cv8 /opt/1cv8
ENV PATH="/opt/1cv8/x86_64/current:${PATH}"

# 1C EDT (опционально)
COPY --from=1c-edt /opt/1C/1CE /opt/1C/1CE

# SonarQube Scanner
RUN curl -L https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-7.2.0.5079-linux-x64.zip -o /tmp/scanner.zip \
    && unzip /tmp/scanner.zip -d /opt \
    && mv /opt/sonar-scanner-* /opt/sonar-scanner \
    && rm /tmp/scanner.zip
ENV PATH="/opt/sonar-scanner/bin:${PATH}"

# apk-ci
COPY --from=builder /app/apk-ci /usr/local/bin/apk-ci
RUN chmod +x /usr/local/bin/apk-ci

# Рабочая директория
WORKDIR /tmp/benadis
VOLUME ["/tmp/benadis"]

# Точка входа для Gitea Act Runner
# Act Runner будет запускать apk-ci через exec
ENTRYPOINT ["/usr/local/bin/apk-ci"]
```

### 9.3 Мониторинг и наблюдаемость

```yaml
# Опциональный стек мониторинга для Swarm
# monitoring-stack.yml

version: "3.8"

services:
  prometheus:
    image: prom/prometheus:v2.48.0
    volumes:
      - prometheus_data:/prometheus
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    deploy:
      placement:
        constraints:
          - node.role == manager

  grafana:
    image: grafana/grafana:10.2.2
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
    ports:
      - "3000:3000"
    deploy:
      placement:
        constraints:
          - node.role == manager

  node-exporter:
    image: prom/node-exporter:v1.7.0
    deploy:
      mode: global

volumes:
  prometheus_data:
  grafana_data:
```

---

## Заключение

### Итоговая рекомендация

Для проекта **apk-ci** рекомендуется использовать **Docker Swarm** как платформу оркестрации по следующим причинам:

1. ✅ **Соответствие профилю нагрузки** — batch CI/CD jobs, не микросервисы
2. ✅ **Простота интеграции с LAN** — прямой доступ к RAC, MSSQL, 1C серверам
3. ✅ **Низкий порог входа** — команда уже знает Docker
4. ✅ **Минимальные операционные затраты** — 0.3 FTE vs 1.0 FTE
5. ✅ **Быстрое внедрение** — 1 неделя vs 1 месяц

### Когда пересмотреть решение

Переход на Kubernetes следует рассмотреть при:

- Масштабировании до 50+ параллельных runners
- Появлении требований к мультитенантности
- Формировании выделенной DevOps команды (2+ FTE)
- Стратегическом решении о cloud-native трансформации

---

**Документ подготовлен:** 2025-12-25
**Следующий пересмотр:** 2026-06-25
