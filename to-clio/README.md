# to-clio

Это конвертер данных из бэкапа frf-saver-а в формат, который понимает [Clio](https://github.com/zverok/clio).

Скачать бинарники можно тут: https://github.com/davidmz/frf-saver/releases

## Как пользоваться?

    Usage:
      to-clio [options] FROM_DIR TO_DIR
      to-clio -h | --help

    Options:
      -h, --help  Show this help and exit
      -ll LEVEL   Log level [default: info]

    FROM_DIR - directory with frf-saver backup (individual feed or all feeds)
    TO_DIR   - target directory ('result' in Clio)

`TO_DIR` — это каталог, в который складывает свои данные Clio. По умолчанию (для Clio) это '<каталог Clio>/result'. Если вы хотите конвертировать фид пользователя `username`, то нужно написать:

    to-clio /.../frf-save/username /.../result

В результате в папке `/.../result` появится подпапка `username` с нужными данными. 

Можно указывать как исходную и общую папку, в которой лежат скачанные фиды (`to-clio /.../frf-save /.../result`) — в этом случае конвертер сконвертирует все подпапки этой папки.

После этого надо запустить Clio вот так:

    ruby bin/clio.rb -i -p /.../result -f username

Если вы использовали стандартный каталог Clio, то `-p /.../result` можно опустить:

    ruby bin/clio.rb -i -f username

Clio немного странно работает с обратными слэшами в путях Windows, поэтому в параметре `-p /.../result` лучше заменить все обратные слэши на прямые.

После того как Clio закончит работу, в `/.../result/username` появятся все нужные для просмотра HTML-файлы.

## Как собрать самому?

Поставить [Go](https://golang.org/doc/install) и сказать:

`go get github.com/davidmz/frf-saver/to-clio`

