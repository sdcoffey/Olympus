import {Component, OnInit, Input} from 'angular2/core';
import {FileInfo} from '../../models/fileinfo';
import {ApiClient} from '../../services/apiclient';

@Component({
  selector: 'file-row',
  templateUrl: 'app/components/filelist/filerow.html',
  styleUrls: ['app/components/filelist/filerow.css']
})
class FileRow implements OnInit {
  @Input() obj: FileInfo

  ngOnInit() {
    if (this.obj.Mode > 0) {
      this.obj.Name = this.obj.Name + "/";
    }
  }
}

@Component({
  selector: 'file-list',
  templateUrl: 'app/components/filelist/filelist.html',
  directives: [FileRow]
})
export class FileListComponent {
  @Input() parentId: string
  children: FileInfo[]

  constructor(private _api: ApiClient) {}

  updateChildren(id: string) {
    this.parentId = id;
    this._api.listFiles(this.parentId)
      .subscribe((children: FileInfo[]) => {
        this.children = children;
      });
  }

  ngOnInit() {
    this.updateChildren('rootNode');
  }

  onChildSelected(fi: FileInfo) {
    if (fi.Mode > 0) {
      this.updateChildren(fi.Id);
    }
  }
}
