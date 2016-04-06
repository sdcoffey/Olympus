import {Component, OnInit, Input} from 'angular2/core';
import {FileInfo} from '../../models/fileinfo';
import {ApiClient} from '../../services/apiclient';
import {Router, RouteParams} from 'angular2/router';

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
export class FileListComponent implements OnInit {
  @Input() parentId: string
  children: FileInfo[]

  constructor(
    private _api: ApiClient,
    private _routeParams: RouteParams,
    private _router: Router
  ) {}

  ngOnInit() {
    this.updateChildren(this._routeParams.get('parentId'));
  }

  updateChildren(id: string) {
    this.parentId = id;
    this._api.listFiles(this.parentId)
      .subscribe((children: FileInfo[]) => {
        this.children = children;
      });
  }

  onChildSelected(fi: FileInfo) {
    if (fi.Mode > 0) {
      this._router.navigate(['/Browse', {parentId: fi.Id}]);
    }
  }
}
