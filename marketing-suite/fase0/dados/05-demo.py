# 05-demo.py | spec secao 3, F0-037 (PROPOSTA F0-P6) | ensaio por padrao; exclui so com MS_APLICAR=1
import os
from django.utils import timezone
from plane.bgtasks.deletion_task import soft_delete_related_objects
from plane.db.models import (Project, User, Cycle, CycleIssue, Module, ModuleIssue,
                             IssueView, Label, IssueLabel, ProjectPage)

APLICAR = os.environ.get("MS_APLICAR") == "1"
projeto = Project.objects.get(workspace__slug="marketing-suite", identifier="MS")
bot = User.objects.get(is_bot=True, bot_type="WORKSPACE_SEED", username=f"bot_user_{projeto.workspace_id}")
print("MODO", "APLICAR" if APLICAR else "ENSAIO", "projeto", projeto.id, "bot", bot.id)

def editado(obj):
    return obj.updated_by_id not in (None, bot.id)

def excluir(obj):
    # igual a SoftDeleteModel.delete, sem o save() que zera created_by e updated_by no shell
    modelo = type(obj)
    modelo.all_objects.filter(pk=obj.pk).update(deleted_at=timezone.now())
    soft_delete_related_objects.delay(obj._meta.app_label, obj._meta.model_name, obj.pk)

def tratar(tipo, obj, em_uso):
    if em_uso or editado(obj):
        print("MANTIDO", tipo, obj.id, obj.name)
        return
    print("EXCLUIR", tipo, obj.id, obj.name)
    if APLICAR:
        excluir(obj)

for c in Cycle.objects.filter(project=projeto, created_by=bot):
    tratar("ciclo", c, CycleIssue.objects.filter(cycle=c, issue__deleted_at__isnull=True).exists())
for m in Module.objects.filter(project=projeto, created_by=bot):
    tratar("modulo", m, ModuleIssue.objects.filter(module=m, issue__deleted_at__isnull=True).exists())
for l in Label.objects.filter(project=projeto, created_by=bot):
    tratar("etiqueta", l, IssueLabel.objects.filter(label=l, issue__deleted_at__isnull=True).exists())
for v in IssueView.objects.filter(project=projeto, created_by=bot):
    tratar("visao", v, False)
for pp in ProjectPage.objects.filter(project=projeto, page__created_by=bot,
                                     page__deleted_at__isnull=True).select_related("page"):
    tratar("pagina", pp.page, False)

DEMO_TEXTO = "Welcome to the Plane Demo Project"
DEMO_CAPA = "https://images.unsplash.com/photo-1691230995681"
campos = {}
if (projeto.description or "").startswith(DEMO_TEXTO):
    campos["description"] = ""
if (projeto.cover_image or "").startswith(DEMO_CAPA):
    campos["cover_image"] = None
print("PROJETO", projeto.id, "limpar:", sorted(campos) or "nada")
if APLICAR and campos:
    Project.objects.filter(id=projeto.id).update(**campos)
