package fileops

type FileOperator struct{}

func NewFileOperator() *FileOperator {
	return &FileOperator{}
}

func (fo *FileOperator) FindFiles(pattern string, options Options) ([]string, error) {
	return FindFiles(pattern, options)
}

func (fo *FileOperator) ReadFile(path string) (string, error) {
	return ReadFile(path)
}

func (fo *FileOperator) GetBasicFileInfo(path string, content string) (FileInfo, error) {
	return GetBasicFileInfo(path, content)
}

func (fo *FileOperator) GetExtendedFileInfo(path string, content string) (FileInfo, error) {
	return GetExtendedFileInfo(path, content)
}
