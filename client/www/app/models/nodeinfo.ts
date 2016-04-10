const _mode_dir: number = (1 << (32 - 1))

export class NodeInfo {
  Id: string;
  Name: string;
  Size: number;
  Mode: number;

  constructor(_id: string, _name: string, _size: number, _mode: number) {
    this.Id = _id;
    this.Name = _name;
    this.Size = _size;
    this.Mode = _mode;
  }

  public isDir(): boolean {
    return (this.Mode & _mode_dir) != 0;
  }

  public iconString(): string {
    if (this.isDir()) {
      return "folder"
    } else {
      return "description"
    }
  }
}
