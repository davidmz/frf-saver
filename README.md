# frf-saver

Очередная бэкапилка френдфида, написана для моего личного пользования.

Скачать бинарники можно тут: https://github.com/davidmz/frf-saver/releases

## Как пользоваться?

    Usage of frf-saver:
      -u="": username to login
      -k="": remote key (see https://friendfeed.com/account/api)
      -d="./frf-save": directory to save data
      -f="": feed name to load (your username if not setted)
      -a=false: save 'username' and all his/her subscriptions (-f ignored)
      -ll="info": log level

Флаги `-u` и `-k` — обязательные, это [юзернейм и ключ](https://friendfeed.com/account/api), которые используются для авторизации в friendfeed API.

`-d` — каталог, в котором будут сохраняться данные. Каждый скачиваемый фид сохраняется в своём подкаталоге и полностью автономен.

`-f` — фид (логин юзера или коммьюнити), который вы хотите скачать. Если он не задан, то скачивается тот же юзер, что задан во флаге `-u`.

`-a` — скачать все фиды, на которые подписан юзер `-u` (и его самого). Если пи этом задан и флаг `-f`, то он игнорируется.

`-ll` — уровень сообщений лога, выводимых на экран. Допустимые значения — trace, debug, info, warn, error, fatal. Менять не имеет особого смысла.

## Что получается в результате?

В результате получается куча технических файлов в папке фида. Там будут json-файлы собственно записей из фида, медиа-файлы (картинки, mp3 и прочие), прикреплённые к постам и аватарки комментирующих и лайкающийх.

### Как это может посмотреть обычный человек?

Пока никак. Может быть, я допишу веб-интерфейс для просмотра архива. Тем не менее, в архиве сохраняется _вся_ информация, которая была в фиде, в том же формате, в котором её отдаёт API френдфида. Так что любая система воостановления френдфида должна, после минимальных доработок, её понять.

Медиа-файлы снабжены расширениями (только картинки и mp3), так что их можно просматривать и в обычном проводнике.

### Совместим ли архив с [Clio](https://github.com/zverok/clio)?

Нет. Но архив можно сконвертировать в формат Clio при помощи утилитки: https://github.com/davidmz/frf-saver/blob/master/to-clio/README.md

После конвертации надо запустить Clio, чтобы она сгенерировала HTML-файлы.

## Как собрать самому?

Поставить [Go](https://golang.org/doc/install) и сказать:

`go get github.com/davidmz/frf-saver`

